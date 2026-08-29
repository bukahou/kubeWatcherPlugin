"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getPodDetail } from "@/datasource/cluster";
import { useClusterStore } from "@/store/clusterStore";
import { useI18n } from "@/i18n/context";
import type { PodDetail } from "@/types/cluster";
import { OverviewTab, ContainersTab, VolumesTab, NetworkTab, SchedulingTab } from "./PodDetailTabs";
import { Box, Container, HardDrive, Network, Settings, History } from "lucide-react";
import { ResourceEventsTab } from "@/components/common/ResourceEventsTab";

interface PodDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  podName: string;
  onViewLogs: (containerName: string) => void;
}

type TabType = "overview" | "containers" | "volumes" | "network" | "scheduling" | "events";

export function PodDetailModal({
  isOpen,
  onClose,
  namespace,
  podName,
  onViewLogs,
}: PodDetailModalProps) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<PodDetail | null>(null);
  const { currentClusterId } = useClusterStore();

  const fetchDetail = useCallback(async () => {
    if (!namespace || !podName || !currentClusterId) return;
    setLoading(true);
    setError("");
    try {
      const res = await getPodDetail({
        ClusterID: currentClusterId,
        Namespace: namespace,
        PodName: podName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [namespace, podName, currentClusterId, t.common.loadFailed]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.pod.overview, icon: <Box className="w-4 h-4" /> },
    { key: "containers", label: t.pod.containers, icon: <Container className="w-4 h-4" /> },
    { key: "volumes", label: t.pod.volumeMounts, icon: <HardDrive className="w-4 h-4" /> },
    { key: "network", label: t.pod.network, icon: <Network className="w-4 h-4" /> },
    { key: "scheduling", label: t.pod.scheduling, icon: <Settings className="w-4 h-4" /> },
    { key: "events", label: t.nav.event, icon: <History className="w-4 h-4" /> },
  ];

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={`Pod: ${podName}`}
      size="xl"
    >
      {loading ? (
        <div className="py-12">
          <LoadingSpinner />
        </div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-full">
          {/* Tabs */}
          <div className="flex border-b border-[var(--border-color)] px-4 shrink-0">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.key
                    ? "border-primary text-primary"
                    : "border-transparent text-muted hover:text-default"
                }`}
              >
                {tab.icon}
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-auto p-6">
            {activeTab === "overview" && <OverviewTab detail={detail} t={t} />}
            {activeTab === "containers" && (
              <ContainersTab containers={detail.containers} onViewLogs={onViewLogs} t={t} />
            )}
            {activeTab === "volumes" && <VolumesTab volumes={detail.volumes || []} t={t} />}
            {activeTab === "network" && <NetworkTab detail={detail} t={t} />}
            {activeTab === "scheduling" && <SchedulingTab detail={detail} t={t} />}
            {activeTab === "events" && (
              <ResourceEventsTab clusterId={currentClusterId} kind="Pod" namespace={detail.namespace} name={detail.name} />
            )}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}
