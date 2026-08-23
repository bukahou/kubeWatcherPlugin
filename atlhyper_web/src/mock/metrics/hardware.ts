/**
 * 硬件健康矩阵 — Mock 数据
 *
 * 对应 MOCK_NODES 的三个节点。字段 1:1 对齐 Master 的 HardwareHealthResponse：
 * 判定结果（status）直接写死，前端不做阈值比较 —— 与真实 API 行为一致。
 * jegan-worker-02 故意造成 warn，用来验证配色和排序。
 */

import type { HardwareHealth } from "@/types/hardware";

export const MOCK_HARDWARE: HardwareHealth = {
  rows: [
    {
      nodeName: "raspi-nfs",
      profile: "raspi4",
      profileLabel: "Raspberry Pi 4",
      cpuUsage: { value: 32.5, status: "good" },
      memUsage: { value: 37.5, status: "good" },
      diskUsage: { value: 62.4, status: "good" },
      cpuTemp: { value: 58.2, max: 80, crit: 85, label: "cpu_thermal", status: "good" },
      diskTemp: null,
      otherTemp: null,
      undervolt: { alarm: false, status: "good" },
      fan: null,
      cpuFreq: { currentGHz: 1.5, maxGHz: 1.5, ratioPct: 100, status: "good" },
      diskAwait: { valueMs: 8.4, device: "mmcblk0", status: "good" },
      sensors: [
        { label: "cpu_thermal", sensor: "temp1", class: "cpu", value: 58.2, max: 80, crit: 85, status: "good" },
      ],
      overall: "good",
    },
    {
      nodeName: "jegan-worker-01",
      profile: "raspi5",
      profileLabel: "Raspberry Pi 5",
      cpuUsage: { value: 28.4, status: "good" },
      memUsage: { value: 52.3, status: "good" },
      diskUsage: { value: 45.8, status: "good" },
      cpuTemp: { value: 61.0, max: 80, crit: 85, label: "cpu_thermal", status: "good" },
      diskTemp: { value: 49.9, max: 82.85, crit: 84.85, label: "nvme0", status: "good" },
      otherTemp: { value: 61.3, max: 85, crit: 95, label: "rp1_adc", status: "good" },
      undervolt: { alarm: false, status: "good" },
      fan: { rpm: 5423, state: 2, maxState: 4, status: "good" },
      cpuFreq: { currentGHz: 1.5, maxGHz: 2.4, ratioPct: 62.5, status: "good" },
      diskAwait: { valueMs: 1.1, device: "nvme0n1", status: "good" },
      sensors: [
        { label: "cpu_thermal", sensor: "temp1", class: "cpu", value: 61.0, max: 80, crit: 85, status: "good" },
        { label: "nvme0", sensor: "temp1", class: "disk", value: 49.9, max: 82.85, crit: 84.85, status: "good" },
        { label: "nvme0", sensor: "temp2", class: "disk", value: 49.9, max: 70, crit: 80, status: "good" },
        { label: "rp1_adc", sensor: "temp1", class: "other", value: 61.3, max: 85, crit: 95, status: "good" },
      ],
      overall: "good",
    },
    {
      nodeName: "jegan-worker-02",
      profile: "raspi5",
      profileLabel: "Raspberry Pi 5",
      cpuUsage: { value: 88.2, status: "warn" },
      memUsage: { value: 71.6, status: "good" },
      diskUsage: { value: 84.1, status: "warn" },
      cpuTemp: { value: 81.4, max: 80, crit: 85, label: "cpu_thermal", status: "warn" },
      diskTemp: { value: 71.2, max: 82.85, crit: 84.85, label: "nvme0", status: "good" },
      otherTemp: { value: 64.8, max: 85, crit: 95, label: "rp1_adc", status: "good" },
      undervolt: { alarm: false, status: "good" },
      fan: { rpm: 6100, state: 4, maxState: 4, status: "good" },
      cpuFreq: { currentGHz: 1.2, maxGHz: 2.4, ratioPct: 50, status: "warn" },
      diskAwait: { valueMs: 62.5, device: "nvme0n1", status: "warn" },
      sensors: [
        { label: "cpu_thermal", sensor: "temp1", class: "cpu", value: 81.4, max: 80, crit: 85, status: "warn" },
        { label: "nvme0", sensor: "temp1", class: "disk", value: 71.2, max: 82.85, crit: 84.85, status: "good" },
        { label: "rp1_adc", sensor: "temp1", class: "other", value: 64.8, max: 85, crit: 95, status: "good" },
      ],
      overall: "warn",
    },
  ],
  summary: {
    maxTemp: { value: 81.4, nodeName: "jegan-worker-02", sensor: "cpu_thermal", status: "warn" },
    maxDiskTemp: { value: 71.2, nodeName: "jegan-worker-02", sensor: "nvme0", status: "good" },
    undervoltNodes: 0,
    throttledNodes: 1,
    maxDiskAwait: { valueMs: 62.5, nodeName: "jegan-worker-02", device: "nvme0n1", status: "warn" },
  },
};

/** 获取硬件健康矩阵 */
export function mockGetHardwareHealth(): HardwareHealth {
  return MOCK_HARDWARE;
}
