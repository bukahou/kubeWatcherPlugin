package query

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/metrics"
)

// metricsRepository Metrics 查询仓库
type metricsRepository struct {
	client   sdk.ClickHouseClient
	nodeRepo repository.NodeRepository

	// IP → NodeName 缓存
	ipMapMu   sync.RWMutex
	ipMap     map[string]string
	ipMapTime time.Time
	ipMapTTL  time.Duration
}

// NewMetricsQueryRepository 创建 Metrics 查询仓库
//
// nodeRepo 用于 IP→NodeName 映射（K8s Node 的 InternalIP → Node.Name）
func NewMetricsQueryRepository(client sdk.ClickHouseClient, nodeRepo repository.NodeRepository) repository.MetricsQueryRepository {
	return &metricsRepository{
		client:   client,
		nodeRepo: nodeRepo,
		ipMapTTL: 5 * time.Minute,
	}
}

// ListAllNodeMetrics 获取所有节点的指标快照
func (r *metricsRepository) ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error) {
	ipMap, err := r.getIPMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("get IP map: %w", err)
	}

	// 获取所有有数据的节点 IP
	ips, err := r.listActiveNodeIPs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active nodes: %w", err)
	}

	var result []metrics.NodeMetrics
	for _, ip := range ips {
		nodeName := ipMap[ip]
		if nodeName == "" {
			nodeName = ip
		}
		nm, err := r.buildNodeMetrics(ctx, ip, nodeName)
		if err != nil {
			continue // 跳过失败的节点
		}
		result = append(result, *nm)
	}
	if result == nil {
		result = []metrics.NodeMetrics{}
	}
	return result, nil
}

// GetNodeMetrics 获取单节点指标快照
func (r *metricsRepository) GetNodeMetrics(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error) {
	ip, err := r.resolveNodeIP(ctx, nodeName)
	if err != nil {
		return nil, err
	}
	return r.buildNodeMetrics(ctx, ip, nodeName)
}

// GetNodeMetricsSeries 获取单指标时序数据
func (r *metricsRepository) GetNodeMetricsSeries(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error) {
	ip, err := r.resolveNodeIP(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	sec := sinceSeconds(since)

	// 根据指标类型判断查询方式
	switch {
	case isCounterMetric(metric):
		return r.queryCounterSeries(ctx, ip, metric, sec)
	default:
		return r.queryGaugeSeries(ctx, ip, metric, sec)
	}
}

// GetMetricsSummary 获取集群节点指标概览
func (r *metricsRepository) GetMetricsSummary(ctx context.Context) (*metrics.Summary, error) {
	all, err := r.ListAllNodeMetrics(ctx)
	if err != nil {
		return nil, err
	}

	s := &metrics.Summary{
		TotalNodes:  len(all),
		OnlineNodes: len(all),
	}
	if len(all) == 0 {
		return s, nil
	}

	var sumCPU, sumMem, maxCPU, maxMem, maxTemp float64
	for _, nm := range all {
		sumCPU += nm.CPU.UsagePct
		sumMem += nm.Memory.UsagePct
		if nm.CPU.UsagePct > maxCPU {
			maxCPU = nm.CPU.UsagePct
		}
		if nm.Memory.UsagePct > maxMem {
			maxMem = nm.Memory.UsagePct
		}
		if nm.Temperature.CPUTempC > maxTemp {
			maxTemp = nm.Temperature.CPUTempC
		}
	}

	n := float64(len(all))
	s.AvgCPUPct = roundTo(sumCPU/n, 2)
	s.AvgMemPct = roundTo(sumMem/n, 2)
	s.MaxCPUPct = roundTo(maxCPU, 2)
	s.MaxMemPct = roundTo(maxMem, 2)
	s.MaxCPUTemp = roundTo(maxTemp, 1)

	return s, nil
}

// =============================================================================
// IP → NodeName 映射
// =============================================================================

// getIPMap 获取 IP→NodeName 映射（带缓存）
func (r *metricsRepository) getIPMap(ctx context.Context) (map[string]string, error) {
	r.ipMapMu.RLock()
	if r.ipMap != nil && time.Since(r.ipMapTime) < r.ipMapTTL {
		m := r.ipMap
		r.ipMapMu.RUnlock()
		return m, nil
	}
	r.ipMapMu.RUnlock()

	r.ipMapMu.Lock()
	defer r.ipMapMu.Unlock()

	// Double-check
	if r.ipMap != nil && time.Since(r.ipMapTime) < r.ipMapTTL {
		return r.ipMap, nil
	}

	nodes, err := r.nodeRepo.List(ctx, model.ListOptions{})
	if err != nil {
		return nil, err
	}

	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		if n.Addresses.InternalIP != "" {
			m[n.Addresses.InternalIP] = n.GetName()
		}
	}

	r.ipMap = m
	r.ipMapTime = time.Now()
	return m, nil
}

// resolveNodeIP 从 NodeName 反查 IP
func (r *metricsRepository) resolveNodeIP(ctx context.Context, nodeName string) (string, error) {
	ipMap, err := r.getIPMap(ctx)
	if err != nil {
		return "", err
	}
	for ip, name := range ipMap {
		if name == nodeName {
			return ip, nil
		}
	}
	// 如果找不到，尝试直接用 nodeName 当 IP
	return nodeName, nil
}

// listActiveNodeIPs 从 ClickHouse 获取最近有数据的节点 IP
func (r *metricsRepository) listActiveNodeIPs(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT ResourceAttributes['net.host.name'] AS ip
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_load1'
		  AND TimeUnix >= now() - INTERVAL 5 MINUTE
	`
	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

// =============================================================================
// 节点指标组装
// =============================================================================

// buildNodeMetrics 组装单节点完整指标
func (r *metricsRepository) buildNodeMetrics(ctx context.Context, ip, nodeName string) (*metrics.NodeMetrics, error) {
	nm := &metrics.NodeMetrics{
		NodeName:  nodeName,
		NodeIP:    ip,
		Timestamp: time.Now(),
	}

	// hwmon chip 可读名只有一张小表，先同步查一次，温度 / 硬件 / 画像三处共用
	chipNameByPath := r.queryChipNames(ctx, ip)
	chipNames := make([]string, 0, len(chipNameByPath))
	for _, name := range chipNameByPath {
		chipNames = append(chipNames, name)
	}
	var machine string // uname machine，Wait 后与 chipNames 一起推导画像

	// 并行查询各指标类别
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	recordErr := func(err error) {
		if err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}
	}

	wg.Add(11)

	// CPU
	go func() {
		defer wg.Done()
		r.fillCPU(ctx, ip, nm)
	}()

	// Memory
	go func() {
		defer wg.Done()
		r.fillMemory(ctx, ip, nm)
	}()

	// Disk
	go func() {
		defer wg.Done()
		r.fillDisks(ctx, ip, nm)
	}()

	// Network
	go func() {
		defer wg.Done()
		r.fillNetworks(ctx, ip, nm)
	}()

	// Temperature
	go func() {
		defer wg.Done()
		r.fillTemperature(ctx, ip, nm, chipNameByPath)
	}()

	// Hardware（风扇 / 散热 / 欠压）
	go func() {
		defer wg.Done()
		r.fillHardware(ctx, ip, nm)
	}()

	// PSI
	go func() {
		defer wg.Done()
		r.fillPSI(ctx, ip, nm)
	}()

	// TCP
	go func() {
		defer wg.Done()
		r.fillTCP(ctx, ip, nm)
	}()

	// System
	go func() {
		defer wg.Done()
		r.fillSystem(ctx, ip, nm)
	}()

	// VMStat + Softnet
	go func() {
		defer wg.Done()
		r.fillVMStat(ctx, ip, nm)
	}()

	// Uptime + Kernel + machine
	go func() {
		defer wg.Done()
		m, err := r.fillSystemInfo(ctx, ip, nm)
		machine = m
		recordErr(err)
	}()

	wg.Wait()

	nm.HardwareProfile = metrics.DetectHardwareProfile(machine, chipNames)
	ensureNodeMetricsSlices(nm)

	return nm, firstErr
}

// ensureNodeMetricsSlices 把 nil 切片补成空切片。
//
// 前端类型声明的是数组（disks: NodeDisk[]），而 Go 的 nil 切片会序列化成 null 而不是 []，
// 于是 n.disks.find(...) 直接抛 TypeError，整页白屏 ——
// 2026-08-24 线上就是这么崩的：两个节点的 fill* 全部没拿到数据（ClickHouse 查询超时），
// 其余五个节点正常，页面却整个挂掉。
//
// 单个查询失败只该让那一格显示「无数据」，不该炸掉整个页面。
// 这是后端的责任：契约说是数组，就必须给数组。
func ensureNodeMetricsSlices(nm *metrics.NodeMetrics) {
	if nm.Disks == nil {
		nm.Disks = []metrics.NodeDisk{}
	}
	if nm.Networks == nil {
		nm.Networks = []metrics.NodeNetwork{}
	}
	if nm.Temperature.Sensors == nil {
		nm.Temperature.Sensors = []metrics.TempSensor{}
	}
	if nm.CPU.FreqHz == nil {
		nm.CPU.FreqHz = []float64{}
	}
	if nm.Hardware != nil {
		if nm.Hardware.Fans == nil {
			nm.Hardware.Fans = []metrics.FanSensor{}
		}
		if nm.Hardware.Cooling == nil {
			nm.Hardware.Cooling = []metrics.CoolingDevice{}
		}
	}
}

// fillCPU 填充 CPU 指标
func (r *metricsRepository) fillCPU(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	// CPU usage from rate of node_cpu_seconds_total
	query := `
		SELECT Attributes['mode'] AS mode, ` + CounterRateExpr + ` AS rate
		FROM otel_metrics_sum
		WHERE MetricName = 'node_cpu_seconds_total'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 5 MINUTE
		GROUP BY Attributes['cpu'], mode
		HAVING count() >= 2
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	modeRates := make(map[string]float64)
	var totalRate float64
	var coreCount int
	seen := make(map[string]bool)

	for rows.Next() {
		var mode string
		var rate float64
		if err := rows.Scan(&mode, &rate); err != nil {
			continue
		}
		modeRates[mode] += rate
		totalRate += rate
		if !seen[mode] {
			seen[mode] = true
		}
	}

	// Count unique cores from idle mode
	coreQuery := `
		SELECT count(DISTINCT Attributes['cpu'])
		FROM otel_metrics_sum
		WHERE MetricName = 'node_cpu_seconds_total'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 5 MINUTE
	`
	r.client.QueryRow(ctx, coreQuery, ip).Scan(&coreCount)

	if totalRate > 0 {
		nm.CPU.UsagePct = roundTo(clamp((1-modeRates["idle"]/totalRate)*100, 0, 100), 2)
		nm.CPU.UserPct = roundTo(safeDivPct(modeRates["user"], totalRate), 2)
		nm.CPU.SystemPct = roundTo(safeDivPct(modeRates["system"], totalRate), 2)
		nm.CPU.IOWaitPct = roundTo(safeDivPct(modeRates["iowait"], totalRate), 2)
	}
	nm.CPU.Cores = coreCount

	// Load averages (gauge)
	loadQuery := `
		SELECT MetricName, argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_load1', 'node_load5', 'node_load15')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY MetricName
	`
	loadRows, err := r.client.Query(ctx, loadQuery, ip)
	if err != nil {
		return
	}
	defer loadRows.Close()

	for loadRows.Next() {
		var name string
		var val float64
		if err := loadRows.Scan(&name, &val); err != nil {
			continue
		}
		switch name {
		case "node_load1":
			nm.CPU.Load1 = roundTo(val, 2)
		case "node_load5":
			nm.CPU.Load5 = roundTo(val, 2)
		case "node_load15":
			nm.CPU.Load15 = roundTo(val, 2)
		}
	}

	// 各核当前频率 + 标称最高频率（cpufreq collector；热降频判定在 Master）
	//
	// 窗口取 5 分钟而非 2 分钟：实测偶发某个节点这一格变成「无数据」，
	// 每次还换一台 —— ClickHouse 瞬时抖动而已，数据本身一直在。
	// argMax 取的仍是最新值，放宽窗口不影响准确性。
	freqRows, err := r.client.Query(ctx, `
		SELECT toUInt32OrZero(Attributes['cpu']) AS cpu,
		       argMaxIf(Value, TimeUnix, MetricName='node_cpu_scaling_frequency_hertz') AS cur,
		       argMaxIf(Value, TimeUnix, MetricName='node_cpu_scaling_frequency_max_hertz') AS maxv
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_cpu_scaling_frequency_hertz', 'node_cpu_scaling_frequency_max_hertz')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 5 MINUTE
		GROUP BY cpu
		ORDER BY cpu
	`, ip)
	if err != nil {
		// 静默失败会让页面显示「无数据」却查不出原因
		logger.Warn("CPU 频率查询失败", "ip", ip, "err", err)
		return
	}
	defer freqRows.Close()
	for freqRows.Next() {
		var cpu uint32
		var cur, maxv float64
		if err := freqRows.Scan(&cpu, &cur, &maxv); err != nil {
			continue
		}
		nm.CPU.FreqHz = append(nm.CPU.FreqHz, cur)
		if maxv > nm.CPU.FreqMaxHz {
			nm.CPU.FreqMaxHz = maxv
		}
	}
}

// fillMemory 填充内存指标
func (r *metricsRepository) fillMemory(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	query := `
		SELECT MetricName, argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName IN (
			'node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes',
			'node_memory_MemFree_bytes', 'node_memory_Cached_bytes',
			'node_memory_Buffers_bytes', 'node_memory_SwapTotal_bytes',
			'node_memory_SwapFree_bytes'
		)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY MetricName
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var val float64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		switch name {
		case "node_memory_MemTotal_bytes":
			nm.Memory.TotalBytes = int64(val)
		case "node_memory_MemAvailable_bytes":
			nm.Memory.AvailableBytes = int64(val)
		case "node_memory_MemFree_bytes":
			nm.Memory.FreeBytes = int64(val)
		case "node_memory_Cached_bytes":
			nm.Memory.CachedBytes = int64(val)
		case "node_memory_Buffers_bytes":
			nm.Memory.BuffersBytes = int64(val)
		case "node_memory_SwapTotal_bytes":
			nm.Memory.SwapTotalBytes = int64(val)
		case "node_memory_SwapFree_bytes":
			nm.Memory.SwapFreeBytes = int64(val)
		}
	}

	if nm.Memory.TotalBytes > 0 {
		nm.Memory.UsagePct = roundTo(clamp(
			(1-float64(nm.Memory.AvailableBytes)/float64(nm.Memory.TotalBytes))*100, 0, 100), 2)
	}
	if nm.Memory.SwapTotalBytes > 0 {
		nm.Memory.SwapUsagePct = roundTo(clamp(
			(1-float64(nm.Memory.SwapFreeBytes)/float64(nm.Memory.SwapTotalBytes))*100, 0, 100), 2)
	}
}

// fillDisks 填充磁盘指标
func (r *metricsRepository) fillDisks(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	// 容量 (gauge)
	query := `
		SELECT Attributes['device'] AS device,
		       Attributes['mountpoint'] AS mp,
		       Attributes['fstype'] AS fs,
		       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_size_bytes') AS total,
		       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_avail_bytes') AS avail,
		       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_files') AS inodes,
		       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_files_free') AS inodesFree,
		       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_readonly') AS readonly
		FROM otel_metrics_gauge
		WHERE MetricName IN (
			'node_filesystem_size_bytes', 'node_filesystem_avail_bytes',
			'node_filesystem_files', 'node_filesystem_files_free', 'node_filesystem_readonly'
		)
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
		  AND Attributes['fstype'] NOT IN ('tmpfs', 'devtmpfs', 'overlay', 'squashfs')
		GROUP BY device, mp, fs
		HAVING total > 0
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	// 一块物理盘可能挂多个分区，容量按行保留（用户看的是挂载点），
	// IO 是设备级的，按归一化后的块设备名索引，稍后回填到它的每个分区行
	var fsDisks []*metrics.NodeDisk
	byBlockDev := make(map[string][]*metrics.NodeDisk)
	for rows.Next() {
		var d metrics.NodeDisk
		var total, avail, inodes, inodesFree, readonly float64
		if err := rows.Scan(&d.Device, &d.MountPoint, &d.FSType, &total, &avail, &inodes, &inodesFree, &readonly); err != nil {
			continue
		}
		d.TotalBytes = int64(total)
		d.AvailBytes = int64(avail)
		if total > 0 {
			d.UsagePct = roundTo(clamp((1-avail/total)*100, 0, 100), 2)
		}
		if inodes > 0 {
			d.InodeUsagePct = roundTo(clamp((1-inodesFree/inodes)*100, 0, 100), 2)
		}
		d.ReadOnly = readonly > 0
		fsDisks = append(fsDisks, &d)
		blk := normalizeBlockDevice(d.Device)
		byBlockDev[blk] = append(byBlockDev[blk], &d)
	}

	// IO rates (sum)
	ioQuery := `
		SELECT Attributes['device'] AS device, MetricName, ` + CounterRateExpr + ` AS rate
		FROM otel_metrics_sum
		WHERE MetricName IN (
			'node_disk_read_bytes_total', 'node_disk_written_bytes_total',
			'node_disk_reads_completed_total', 'node_disk_writes_completed_total',
			'node_disk_io_time_seconds_total',
			'node_disk_read_time_seconds_total', 'node_disk_write_time_seconds_total',
			'node_disk_io_time_weighted_seconds_total'
		)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 5 MINUTE
		GROUP BY device, MetricName
		HAVING count() >= 2
	`
	ioRows, err := r.client.Query(ctx, ioQuery, ip)
	if err != nil {
		return
	}
	defer ioRows.Close()

	// 块设备级 IO 先攒在这里，最后回填到对应的分区行
	type blockIO struct {
		readBytes, writeBytes, readIOPS, writeIOPS, utilPct, queue float64
		readTime, writeTime                                        float64
	}
	blockIOs := make(map[string]*blockIO)

	for ioRows.Next() {
		var device, metricName string
		var rate float64
		if err := ioRows.Scan(&device, &metricName, &rate); err != nil {
			continue
		}
		if !isWholeBlockDevice(device) {
			continue // 分区级记录已包含在整盘行里
		}
		b, ok := blockIOs[device]
		if !ok {
			b = &blockIO{}
			blockIOs[device] = b
		}
		switch metricName {
		case "node_disk_read_bytes_total":
			b.readBytes = rate
		case "node_disk_written_bytes_total":
			b.writeBytes = rate
		case "node_disk_reads_completed_total":
			b.readIOPS = rate
		case "node_disk_writes_completed_total":
			b.writeIOPS = rate
		case "node_disk_io_time_seconds_total":
			b.utilPct = clamp(rate*100, 0, 100)
		case "node_disk_read_time_seconds_total":
			b.readTime = rate
		case "node_disk_write_time_seconds_total":
			b.writeTime = rate
		case "node_disk_io_time_weighted_seconds_total":
			b.queue = rate
		}
	}

	// 回填：同一块盘的所有分区行共享该盘的 IO 读数
	for dev, b := range blockIOs {
		targets := byBlockDev[dev]
		if len(targets) == 0 {
			// 没有任何文件系统挂在它上面（LVM 的 dm-N、未挂载的盘）——
			// 单独成行，让 IO 至少可见，而不是悄悄丢掉
			d := &metrics.NodeDisk{Device: dev}
			fsDisks = append(fsDisks, d)
			targets = []*metrics.NodeDisk{d}
		}
		for _, d := range targets {
			d.ReadBytesPerSec = roundTo(b.readBytes, 2)
			d.WriteBytesPerSec = roundTo(b.writeBytes, 2)
			d.ReadIOPS = roundTo(b.readIOPS, 2)
			d.WriteIOPS = roundTo(b.writeIOPS, 2)
			d.IOUtilPct = roundTo(b.utilPct, 2)
			d.QueueDepth = roundTo(b.queue, 2)
			d.AwaitReadMs = awaitMs(b.readTime, b.readIOPS)
			d.AwaitWriteMs = awaitMs(b.writeTime, b.writeIOPS)
		}
	}

	for _, d := range fsDisks {
		nm.Disks = append(nm.Disks, *d)
	}
}

// fillNetworks 填充网络指标
func (r *metricsRepository) fillNetworks(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	// 网络状态 (gauge)
	query := `
		SELECT Attributes['device'] AS iface,
		       argMaxIf(Value, TimeUnix, MetricName='node_network_up') AS up,
		       argMaxIf(Value, TimeUnix, MetricName='node_network_speed_bytes') AS speed,
		       argMaxIf(Value, TimeUnix, MetricName='node_network_mtu_bytes') AS mtu
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_network_up', 'node_network_speed_bytes', 'node_network_mtu_bytes')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY iface
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	ifMap := make(map[string]*metrics.NodeNetwork)
	for rows.Next() {
		var n metrics.NodeNetwork
		var up, speed, mtu float64
		if err := rows.Scan(&n.Interface, &up, &speed, &mtu); err != nil {
			continue
		}
		n.Up = up > 0
		n.SpeedBps = int64(speed)
		n.MTU = int(mtu)
		ifMap[n.Interface] = &n
	}

	// 网络吞吐 (rate from sum)
	rateQuery := `
		SELECT Attributes['device'] AS iface, MetricName, ` + CounterRateExpr + ` AS rate
		FROM otel_metrics_sum
		WHERE MetricName IN (
			'node_network_receive_bytes_total', 'node_network_transmit_bytes_total',
			'node_network_receive_packets_total', 'node_network_transmit_packets_total',
			'node_network_receive_errs_total', 'node_network_transmit_errs_total',
			'node_network_receive_drop_total', 'node_network_transmit_drop_total'
		)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 5 MINUTE
		GROUP BY iface, MetricName
		HAVING count() >= 2
	`
	rateRows, err := r.client.Query(ctx, rateQuery, ip)
	if err != nil {
		return
	}
	defer rateRows.Close()

	for rateRows.Next() {
		var iface, metricName string
		var rate float64
		if err := rateRows.Scan(&iface, &metricName, &rate); err != nil {
			continue
		}
		n, ok := ifMap[iface]
		if !ok {
			n = &metrics.NodeNetwork{Interface: iface}
			ifMap[iface] = n
		}
		switch metricName {
		case "node_network_receive_bytes_total":
			n.RxBytesPerSec = roundTo(rate, 2)
		case "node_network_transmit_bytes_total":
			n.TxBytesPerSec = roundTo(rate, 2)
		case "node_network_receive_packets_total":
			n.RxPktPerSec = roundTo(rate, 2)
		case "node_network_transmit_packets_total":
			n.TxPktPerSec = roundTo(rate, 2)
		case "node_network_receive_errs_total":
			n.RxErrPerSec = roundTo(rate, 2)
		case "node_network_transmit_errs_total":
			n.TxErrPerSec = roundTo(rate, 2)
		case "node_network_receive_drop_total":
			n.RxDropPerSec = roundTo(rate, 2)
		case "node_network_transmit_drop_total":
			n.TxDropPerSec = roundTo(rate, 2)
		}
	}

	for _, n := range ifMap {
		nm.Networks = append(nm.Networks, *n)
	}
}

// fillTemperature 填充温度指标：所有 hwmon 传感器 + 可读 chip 名。
// CPU 温度 / 阈值取 CPU 类传感器（coretemp / SoC thermal zone）里最热的一个，
// 不再取 Sensors[0]（那取决于 ClickHouse 的返回顺序，NVMe 排前面就会把盘温当 CPU 温）。
func (r *metricsRepository) fillTemperature(ctx context.Context, ip string, nm *metrics.NodeMetrics, chipNameByPath map[string]string) {
	query := `
		SELECT Attributes['chip'] AS chip, Attributes['sensor'] AS sensor,
		       argMaxIf(Value, TimeUnix, MetricName='node_hwmon_temp_celsius') AS current,
		       argMaxIf(Value, TimeUnix, MetricName='node_hwmon_temp_max_celsius') AS maxv,
		       argMaxIf(Value, TimeUnix, MetricName='node_hwmon_temp_crit_celsius') AS crit
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_hwmon_temp_celsius', 'node_hwmon_temp_max_celsius', 'node_hwmon_temp_crit_celsius')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY chip, sensor
		ORDER BY chip, sensor
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	var hottestCPU *metrics.TempSensor
	for rows.Next() {
		var s metrics.TempSensor
		if err := rows.Scan(&s.Chip, &s.Sensor, &s.CurrentC, &s.MaxC, &s.CritC); err != nil {
			continue
		}
		s.ChipName = chipNameByPath[s.Chip]
		s.CurrentC = roundTo(s.CurrentC, 2)
		s.MaxC = sanitizeTempLimit(s.MaxC)
		s.CritC = sanitizeTempLimit(s.CritC)
		nm.Temperature.Sensors = append(nm.Temperature.Sensors, s)
		if s.Class() == metrics.TempClassCPU && (hottestCPU == nil || s.CurrentC > hottestCPU.CurrentC) {
			last := nm.Temperature.Sensors[len(nm.Temperature.Sensors)-1]
			hottestCPU = &last
		}
	}
	if hottestCPU != nil {
		nm.Temperature.CPUTempC = roundTo(hottestCPU.CurrentC, 1)
		nm.Temperature.CPUMaxC = hottestCPU.MaxC
		nm.Temperature.CPUCritC = hottestCPU.CritC
	}
}

// queryChipNames 取 hwmon chip 路径 → 可读名（node_hwmon_chip_names 的 chip / chip_name 标签）。
// 查不到返回空 map，调用方按「无可读名」降级，不报错。
func (r *metricsRepository) queryChipNames(ctx context.Context, ip string) map[string]string {
	names := make(map[string]string)
	rows, err := r.client.Query(ctx, `
		SELECT DISTINCT Attributes['chip'], Attributes['chip_name']
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_hwmon_chip_names'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
	`, ip)
	if err != nil {
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var chip, name string
		if err := rows.Scan(&chip, &name); err == nil && chip != "" {
			names[chip] = name
		}
	}
	return names
}

// fillHardware 填充风扇 / 散热设备 / 欠压告警。
// 任一传感器不存在时对应切片为空 / 指针为 nil，Master 据此显示「无数据」。
func (r *metricsRepository) fillHardware(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	hw := &metrics.NodeHardware{Fans: []metrics.FanSensor{}, Cooling: []metrics.CoolingDevice{}}

	fanRows, err := r.client.Query(ctx, `
		SELECT Attributes['chip'], Attributes['sensor'], argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_hwmon_fan_rpm'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY Attributes['chip'], Attributes['sensor']
	`, ip)
	if err == nil {
		for fanRows.Next() {
			var f metrics.FanSensor
			if err := fanRows.Scan(&f.Chip, &f.Sensor, &f.RPM); err == nil {
				hw.Fans = append(hw.Fans, f)
			}
		}
		fanRows.Close()
	}

	coolRows, err := r.client.Query(ctx, `
		SELECT Attributes['name'] AS name, Attributes['type'] AS type,
		       argMaxIf(Value, TimeUnix, MetricName='node_cooling_device_cur_state') AS cur,
		       argMaxIf(Value, TimeUnix, MetricName='node_cooling_device_max_state') AS maxv
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_cooling_device_cur_state', 'node_cooling_device_max_state')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY name, type
		ORDER BY type, name
	`, ip)
	if err == nil {
		for coolRows.Next() {
			var c metrics.CoolingDevice
			var cur, maxv float64
			if err := coolRows.Scan(&c.Name, &c.Type, &cur, &maxv); err == nil {
				c.CurState, c.MaxState = int(cur), int(maxv)
				hw.Cooling = append(hw.Cooling, c)
			}
		}
		coolRows.Close()
	}

	// 欠压位：只有树莓派 rpi_volt 有；用 count 区分「无传感器」与「值为 0」
	var alarmCount uint64
	var alarm float64
	err = r.client.QueryRow(ctx, `
		SELECT count(), argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_hwmon_in_lcrit_alarm_volts'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
	`, ip).Scan(&alarmCount, &alarm)
	if err == nil && alarmCount > 0 {
		v := alarm > 0
		hw.UndervoltAlarm = &v
	}

	nm.Hardware = hw
}

// 块设备的分区命名有两套规则，不能用一条正则概括：
//   - 基名以数字结尾的（nvme0n1、mmcblk0）用 "p<N>" 分隔：nvme0n1p2
//   - 基名以字母结尾的（sda、vdb）直接接数字：sda2
//
// 用一条 `.*?[a-z]` 之类的松正则会把 nvme0n1 自己切成 nvme0n + 1。
var (
	pPartition = regexp.MustCompile(`^(.+\d)p\d+$`)
	// 只对 sd/hd/vd/xvd 这几族用「字母基名 + 数字」的分区规则。
	// 不能用通用的 ^([a-z]+)\d+$ —— mmcblk0 是整盘不是分区，那条规则会把它切成 mmcblk。
	sdaPartition = regexp.MustCompile(`^((?:x?vd|sd|hd)[a-z]+)\d+$`)
)

// normalizeBlockDevice 把文件系统的设备名归一到 diskstats 用的块设备名。
//
// 两边命名不一致（filesystem: /dev/nvme0n1p2，diskstats: nvme0n1），
// 不归一化就会让同一块盘裂成「有容量无 IO」和「有 IO 无容量」两行。
//
// LVM 例外：/dev/mapper/xxx 在 diskstats 里叫 dm-N，无法从名字推导，保持原样
// —— 那种情况下根分区拿不到 IO，但至少不会张冠李戴。
func normalizeBlockDevice(dev string) string {
	dev = strings.TrimPrefix(dev, "/dev/")
	if dev == "" || strings.HasPrefix(dev, "mapper/") || strings.HasPrefix(dev, "dm-") {
		return dev
	}
	if m := pPartition.FindStringSubmatch(dev); m != nil {
		return m[1]
	}
	if m := sdaPartition.FindStringSubmatch(dev); m != nil {
		return m[1]
	}
	return dev
}

// isWholeBlockDevice 判断是否整盘（而非分区）。
// /proc/diskstats 会同时给出 mmcblk0 与 mmcblk0p1/p2 三行，
// 分区行的读写已经计入整盘行，两者都取会让同一块盘的 IO 重复出现。
func isWholeBlockDevice(dev string) bool {
	return normalizeBlockDevice(dev) == dev
}

// tempLimitCeilingC 超过它的温度阈值视为传感器垃圾值（实测 NVMe temp2 上报 max=65261.85）
const tempLimitCeilingC = 200

// sanitizeTempLimit 把物理上不可能的阈值归零，交给 Master 按画像兜底
func sanitizeTempLimit(v float64) float64 {
	if v <= 0 || v > tempLimitCeilingC {
		return 0
	}
	return v
}

// awaitMs 平均 IO 延迟（毫秒）= 累计耗时速率 ÷ 完成次数速率
func awaitMs(timeRate, opsRate float64) float64 {
	if opsRate <= 0 {
		return 0
	}
	return roundTo(timeRate/opsRate*1000, 2)
}

// fillPSI 填充 Pressure Stall Info
func (r *metricsRepository) fillPSI(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	query := `
		SELECT MetricName, ` + CounterRateExpr + ` AS rate
		FROM otel_metrics_sum
		WHERE MetricName IN (
			'node_pressure_cpu_waiting_seconds_total',
			'node_pressure_memory_waiting_seconds_total',
			'node_pressure_memory_stalled_seconds_total',
			'node_pressure_io_waiting_seconds_total',
			'node_pressure_io_stalled_seconds_total'
		)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 5 MINUTE
		GROUP BY MetricName
		HAVING count() >= 2
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var rate float64
		if err := rows.Scan(&name, &rate); err != nil {
			continue
		}
		pct := roundTo(clamp(rate*100, 0, 100), 2)
		switch name {
		case "node_pressure_cpu_waiting_seconds_total":
			nm.PSI.CPUSomePct = pct
		case "node_pressure_memory_waiting_seconds_total":
			nm.PSI.MemSomePct = pct
		case "node_pressure_memory_stalled_seconds_total":
			nm.PSI.MemFullPct = pct
		case "node_pressure_io_waiting_seconds_total":
			nm.PSI.IOSomePct = pct
		case "node_pressure_io_stalled_seconds_total":
			nm.PSI.IOFullPct = pct
		}
	}
}

// fillTCP 填充 TCP 连接状态
func (r *metricsRepository) fillTCP(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	query := `
		SELECT MetricName, argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName IN (
			'node_netstat_Tcp_CurrEstab',
			'node_sockstat_TCP_alloc',
			'node_sockstat_TCP_inuse',
			'node_sockstat_TCP_tw',
			'node_sockstat_sockets_used'
		)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY MetricName
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var val float64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		switch name {
		case "node_netstat_Tcp_CurrEstab":
			nm.TCP.CurrEstab = int64(val)
		case "node_sockstat_TCP_alloc":
			nm.TCP.Alloc = int64(val)
		case "node_sockstat_TCP_inuse":
			nm.TCP.InUse = int64(val)
		case "node_sockstat_TCP_tw":
			nm.TCP.TimeWait = int64(val)
		case "node_sockstat_sockets_used":
			nm.TCP.SocketsUsed = int64(val)
		}
	}
}

// fillSystem 填充系统指标
func (r *metricsRepository) fillSystem(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	query := `
		SELECT MetricName, argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName IN (
			'node_nf_conntrack_entries', 'node_nf_conntrack_entries_limit',
			'node_filefd_allocated', 'node_filefd_maximum',
			'node_entropy_available_bits',
			'node_procs_running', 'node_procs_blocked',
			'node_arp_entries',
			'node_timex_offset_seconds', 'node_timex_sync_status'
		)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 2 MINUTE
		GROUP BY MetricName
	`
	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var val float64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		switch name {
		case "node_nf_conntrack_entries":
			nm.System.ConntrackEntries = int64(val)
		case "node_nf_conntrack_entries_limit":
			nm.System.ConntrackLimit = int64(val)
		case "node_filefd_allocated":
			nm.System.FilefdAllocated = int64(val)
		case "node_filefd_maximum":
			nm.System.FilefdMax = int64(val)
		case "node_entropy_available_bits":
			nm.System.EntropyBits = int64(val)
		case "node_procs_running":
			nm.System.ProcsRunning = int64(val)
		case "node_procs_blocked":
			nm.System.ProcsBlocked = int64(val)
		case "node_arp_entries":
			// 按网卡上报，多网卡时 argMax 只会留一条；家庭集群每台只有一块业务网卡
			nm.System.ArpEntries = int64(val)
		case "node_timex_offset_seconds":
			nm.System.TimeOffsetMs = roundTo(val*1000, 3)
		case "node_timex_sync_status":
			nm.System.TimeSynced = val > 0
		}
	}
}

// fillVMStat 填充 VMStat + Softnet
// vmstat 指标在 gauge 表（OTel collector 将其归类为 gauge），softnet 在 sum 表
func (r *metricsRepository) fillVMStat(ctx context.Context, ip string, nm *metrics.NodeMetrics) {
	rateQuery := `
		SELECT MetricName, ` + CounterRateExpr + ` AS rate
		FROM %s
		WHERE MetricName IN (%s)
		AND ResourceAttributes['net.host.name'] = ?
		AND TimeUnix >= now() - INTERVAL 5 MINUTE
		GROUP BY MetricName
		HAVING count() >= 2
	`

	// vmstat: gauge 表
	vmstatQuery := fmt.Sprintf(rateQuery, "otel_metrics_gauge",
		"'node_vmstat_pgfault', 'node_vmstat_pgmajfault', 'node_vmstat_pswpin', 'node_vmstat_pswpout'")
	// softnet: sum 表
	softnetQuery := fmt.Sprintf(rateQuery, "otel_metrics_sum",
		"'node_softnet_dropped_total', 'node_softnet_times_squeezed_total'")

	for _, q := range []string{vmstatQuery, softnetQuery} {
		rows, err := r.client.Query(ctx, q, ip)
		if err != nil {
			continue
		}
		for rows.Next() {
			var name string
			var rate float64
			if err := rows.Scan(&name, &rate); err != nil {
				continue
			}
			switch name {
			case "node_vmstat_pgfault":
				nm.VMStat.PgFaultPerSec = roundTo(rate, 2)
			case "node_vmstat_pgmajfault":
				nm.VMStat.PgMajFaultPerSec = roundTo(rate, 2)
			case "node_vmstat_pswpin":
				nm.VMStat.PswpInPerSec = roundTo(rate, 2)
			case "node_vmstat_pswpout":
				nm.VMStat.PswpOutPerSec = roundTo(rate, 2)
			case "node_softnet_dropped_total":
				nm.Softnet.DroppedPerSec = roundTo(rate, 2)
			case "node_softnet_times_squeezed_total":
				nm.Softnet.SqueezedPerSec = roundTo(rate, 2)
			}
		}
		rows.Close()
	}

	// OOM kill 用累计值而非速率：一次 OOM 之后速率立刻归零，
	// 但「这台机器杀过进程」这件事必须一直看得见
	var oomKill float64
	if err := r.client.QueryRow(ctx, `
		SELECT argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_vmstat_oom_kill'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 5 MINUTE
	`, ip).Scan(&oomKill); err == nil {
		nm.Memory.OOMKillTotal = int64(oomKill)
	}
}

// fillSystemInfo 填充 Uptime + Kernel，并返回 uname machine（x86_64 / aarch64，画像识别用）
func (r *metricsRepository) fillSystemInfo(ctx context.Context, ip string, nm *metrics.NodeMetrics) (string, error) {
	// Boot time
	var bootTime float64
	err := r.client.QueryRow(ctx, `
		SELECT argMax(Value, TimeUnix)
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_boot_time_seconds'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL 2 MINUTE
	`, ip).Scan(&bootTime)
	if err == nil && bootTime > 0 {
		nm.Uptime = time.Now().Unix() - int64(bootTime)
	}

	// Kernel + machine from uname info labels
	var kernel, machine string
	err = r.client.QueryRow(ctx, `
		SELECT Attributes['release'], Attributes['machine']
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_uname_info'
		  AND ResourceAttributes['net.host.name'] = ?
		ORDER BY TimeUnix DESC LIMIT 1
	`, ip).Scan(&kernel, &machine)
	if err == nil {
		nm.Kernel = kernel
	}

	return machine, nil
}

// =============================================================================
// 时序查询
// =============================================================================

// queryGaugeSeries gauge 指标时序查询
func (r *metricsRepository) queryGaugeSeries(ctx context.Context, ip, metric string, since int64) ([]metrics.Point, error) {
	query := fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL 60 SECOND) AS ts,
		       avg(Value) AS val
		FROM otel_metrics_gauge
		WHERE MetricName = ?
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ts
		ORDER BY ts
	`, since)

	rows, err := r.client.Query(ctx, query, metric, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []metrics.Point
	for rows.Next() {
		var p metrics.Point
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if points == nil {
		points = []metrics.Point{}
	}
	return points, rows.Err()
}

// queryCounterSeries counter 指标时序查询（Go 侧计算 rate）
func (r *metricsRepository) queryCounterSeries(ctx context.Context, ip, metric string, since int64) ([]metrics.Point, error) {
	query := fmt.Sprintf(`
		SELECT TimeUnix, Value
		FROM otel_metrics_sum
		WHERE MetricName = ?
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		ORDER BY TimeUnix
	`, since)

	rows, err := r.client.Query(ctx, query, metric, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var raw []rawPoint
	for rows.Next() {
		var p rawPoint
		if err := rows.Scan(&p.Time, &p.Value); err != nil {
			return nil, err
		}
		raw = append(raw, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := computeRateSeries(raw)
	if result == nil {
		result = []metrics.Point{}
	}
	return result, nil
}

// isCounterMetric 判断是否 counter 类型指标
func isCounterMetric(metric string) bool {
	counterMetrics := map[string]bool{
		"node_cpu_seconds_total":            true,
		"node_disk_read_bytes_total":        true,
		"node_disk_written_bytes_total":     true,
		"node_disk_reads_completed_total":   true,
		"node_disk_writes_completed_total":  true,
		"node_disk_io_time_seconds_total":   true,
		"node_network_receive_bytes_total":  true,
		"node_network_transmit_bytes_total": true,
		"node_softnet_dropped_total":        true,
		"node_softnet_times_squeezed_total": true,
	}
	return counterMetrics[metric]
}

// GetNodeMetricsHistory 获取节点历史时序（按指标分组: cpu/memory/disk/temp）
//
// 时间粒度自动调整:
//   - ≤6h  → 1min  (最多 360 点)
//   - ≤24h → 5min  (最多 288 点)
//   - >24h → 15min
func (r *metricsRepository) GetNodeMetricsHistory(ctx context.Context, nodeName string, since time.Duration) (map[string][]metrics.Point, error) {
	ip, err := r.resolveNodeIP(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	sec := sinceSeconds(since)

	// 自动调整采样粒度
	intervalSec := 60 // 默认 1 分钟
	if since > 24*time.Hour {
		intervalSec = 900 // 15 分钟
	} else if since > 6*time.Hour {
		intervalSec = 300 // 5 分钟
	}

	result := map[string][]metrics.Point{
		"cpu":    {},
		"memory": {},
		"disk":   {},
		"temp":   {},
	}

	// 并行查询 4 个指标
	type kv struct {
		key    string
		points []metrics.Point
	}
	ch := make(chan kv, 4)
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		pts, _ := r.queryCPUHistory(ctx, ip, sec, intervalSec)
		ch <- kv{"cpu", pts}
	}()
	go func() {
		defer wg.Done()
		pts, _ := r.queryMemoryHistory(ctx, ip, sec, intervalSec)
		ch <- kv{"memory", pts}
	}()
	go func() {
		defer wg.Done()
		pts, _ := r.queryDiskHistory(ctx, ip, sec, intervalSec)
		ch <- kv{"disk", pts}
	}()
	go func() {
		defer wg.Done()
		pts, _ := r.queryTempHistory(ctx, ip, sec, intervalSec)
		ch <- kv{"temp", pts}
	}()

	wg.Wait()
	close(ch)

	for item := range ch {
		if item.points != nil {
			result[item.key] = item.points
		}
	}

	return result, nil
}

// queryCPUHistory 查询 CPU 使用率历史
//
// CPU 是 counter 类型 (node_cpu_seconds_total)，按 mode 分组后计算:
//
//	cpu_usage_pct = (1 - idle_delta / total_delta) * 100
func (r *metricsRepository) queryCPUHistory(ctx context.Context, ip string, sinceSec int64, intervalSec int) ([]metrics.Point, error) {
	query := fmt.Sprintf(`
		SELECT ts, 1 - sumIf(delta, mode = 'idle') / sum(delta) AS cpu_pct
		FROM (
			SELECT Attributes['mode'] AS mode,
			       toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
			       max(Value) - min(Value) AS delta
			FROM otel_metrics_sum
			WHERE MetricName = 'node_cpu_seconds_total'
			  AND ResourceAttributes['net.host.name'] = ?
			  AND TimeUnix >= now() - INTERVAL %d SECOND
			GROUP BY mode, Attributes['cpu'], ts
			HAVING delta > 0
		)
		GROUP BY ts
		HAVING sum(delta) > 0
		ORDER BY ts
	`, intervalSec, sinceSec)

	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []metrics.Point
	for rows.Next() {
		var p metrics.Point
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			continue
		}
		p.Value = roundTo(clamp(p.Value*100, 0, 100), 2)
		points = append(points, p)
	}
	if points == nil {
		points = []metrics.Point{}
	}
	return points, rows.Err()
}

// queryMemoryHistory 查询内存使用率历史
//
// Gauge 类型: usage = (1 - available / total) * 100
func (r *metricsRepository) queryMemoryHistory(ctx context.Context, ip string, sinceSec int64, intervalSec int) ([]metrics.Point, error) {
	query := fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       (1 - avgIf(Value, MetricName = 'node_memory_MemAvailable_bytes') /
		            avgIf(Value, MetricName = 'node_memory_MemTotal_bytes')) * 100 AS mem_pct
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ts
		HAVING avgIf(Value, MetricName = 'node_memory_MemTotal_bytes') > 0
		ORDER BY ts
	`, intervalSec, sinceSec)

	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []metrics.Point
	for rows.Next() {
		var p metrics.Point
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			continue
		}
		p.Value = roundTo(clamp(p.Value, 0, 100), 2)
		points = append(points, p)
	}
	if points == nil {
		points = []metrics.Point{}
	}
	return points, rows.Err()
}

// queryDiskHistory 查询磁盘使用率历史（根分区 /）
//
// Gauge 类型: usage = (1 - avail / total) * 100
func (r *metricsRepository) queryDiskHistory(ctx context.Context, ip string, sinceSec int64, intervalSec int) ([]metrics.Point, error) {
	query := fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       (1 - avgIf(Value, MetricName = 'node_filesystem_avail_bytes') /
		            avgIf(Value, MetricName = 'node_filesystem_size_bytes')) * 100 AS disk_pct
		FROM otel_metrics_gauge
		WHERE MetricName IN ('node_filesystem_size_bytes', 'node_filesystem_avail_bytes')
		  AND ResourceAttributes['net.host.name'] = ?
		  AND Attributes['mountpoint'] = '/'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ts
		HAVING avgIf(Value, MetricName = 'node_filesystem_size_bytes') > 0
		ORDER BY ts
	`, intervalSec, sinceSec)

	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []metrics.Point
	for rows.Next() {
		var p metrics.Point
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			continue
		}
		p.Value = roundTo(clamp(p.Value, 0, 100), 2)
		points = append(points, p)
	}
	if points == nil {
		points = []metrics.Point{}
	}
	return points, rows.Err()
}

// queryTempHistory 查询 CPU 温度历史
//
// Gauge 类型: 取每个时间桶内最大传感器温度
func (r *metricsRepository) queryTempHistory(ctx context.Context, ip string, sinceSec int64, intervalSec int) ([]metrics.Point, error) {
	query := fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       max(Value) AS temp_c
		FROM otel_metrics_gauge
		WHERE MetricName = 'node_hwmon_temp_celsius'
		  AND ResourceAttributes['net.host.name'] = ?
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ts
		ORDER BY ts
	`, intervalSec, sinceSec)

	rows, err := r.client.Query(ctx, query, ip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []metrics.Point
	for rows.Next() {
		var p metrics.Point
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			continue
		}
		p.Value = roundTo(p.Value, 1)
		points = append(points, p)
	}
	if points == nil {
		points = []metrics.Point{}
	}
	return points, rows.Err()
}
