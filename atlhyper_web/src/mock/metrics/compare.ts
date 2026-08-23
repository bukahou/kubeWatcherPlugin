/**
 * 节点对比表 — Mock 数据
 *
 * 与 MOCK_HARDWARE 同源：硬件列的判定必须和硬件矩阵一致，
 * 同一个读数在两个页面显示不同颜色是最难排查的 UI bug。
 */

import type { NodeComparison } from "@/types/hardware";

export const MOCK_COMPARE: NodeComparison = {
  columns: [
    "cpuTemp", "diskTemp", "undervolt", "freqRatio", "diskAwait",
    "diskUsage", "cpu", "psiCpu", "mem", "psiMem", "netErr",
  ],
  rows: [
    {
      nodeName: "raspi-nfs",
      profile: "raspi4",
      cells: {
        cpuTemp: { value: 58.2, text: "58.2°C", status: "good" },
        diskTemp: { value: null, status: "good" },
        undervolt: { value: 0, status: "good" },
        freqRatio: { value: 100, text: "100%", status: "good" },
        diskAwait: { value: 8.4, text: "8.4 ms", status: "good" },
        diskUsage: { value: 62.4, text: "62.4%", status: "good" },
        cpu: { value: 32.5, text: "32.5%", status: "good" },
        psiCpu: { value: 4.2, text: "4.2%", status: "good" },
        mem: { value: 37.5, text: "37.5%", status: "good" },
        psiMem: { value: 0.8, text: "0.8%", status: "good" },
        netErr: { value: 0, text: "0.00/s", status: "good" },
      },
      overall: "good",
    },
    {
      nodeName: "jegan-worker-01",
      profile: "raspi5",
      cells: {
        cpuTemp: { value: 61.0, text: "61.0°C", status: "good" },
        diskTemp: { value: 49.9, text: "49.9°C", status: "good" },
        undervolt: { value: 0, status: "good" },
        freqRatio: { value: 62.5, text: "63%", status: "good" },
        diskAwait: { value: 1.1, text: "1.1 ms", status: "good" },
        diskUsage: { value: 45.8, text: "45.8%", status: "good" },
        cpu: { value: 28.4, text: "28.4%", status: "good" },
        psiCpu: { value: 2.1, text: "2.1%", status: "good" },
        mem: { value: 52.3, text: "52.3%", status: "good" },
        psiMem: { value: 0.3, text: "0.3%", status: "good" },
        netErr: { value: 0, text: "0.00/s", status: "good" },
      },
      overall: "good",
    },
    {
      nodeName: "jegan-worker-02",
      profile: "raspi5",
      cells: {
        cpuTemp: { value: 81.4, text: "81.4°C", status: "warn" },
        diskTemp: { value: 71.2, text: "71.2°C", status: "good" },
        undervolt: { value: 0, status: "good" },
        freqRatio: { value: 50, text: "50%", status: "warn" },
        diskAwait: { value: 62.5, text: "62.5 ms", status: "warn" },
        diskUsage: { value: 84.1, text: "84.1%", status: "warn" },
        cpu: { value: 88.2, text: "88.2%", status: "warn" },
        psiCpu: { value: 34.5, text: "34.5%", status: "warn" },
        mem: { value: 71.6, text: "71.6%", status: "good" },
        psiMem: { value: 6.2, text: "6.2%", status: "good" },
        netErr: { value: 1.4, text: "1.40/s", status: "warn" },
      },
      overall: "warn",
    },
  ],
};

/** 获取节点对比表 */
export function mockGetNodeComparison(): NodeComparison {
  return MOCK_COMPARE;
}
