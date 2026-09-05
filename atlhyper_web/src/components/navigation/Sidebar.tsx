"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Github } from "lucide-react";
import { useState, useEffect } from "react";
import { useAuthStore } from "@/store/authStore";
import { navGroups, getActiveGroups, loadExpandedGroups, saveExpandedGroups } from "./SidebarTypes";
import { SidebarNavGroup } from "./SidebarNavGroup";
import { SidebarClusterSelector } from "./SidebarClusterSelector";
import { SidebarFooter } from "./SidebarFooter";

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const pathname = usePathname();
  const { isAuthenticated, user } = useAuthStore();
  const isAdmin = user?.role === 3;
  const [expandedGroups, setExpandedGroups] = useState<string[]>(() => loadExpandedGroups(pathname));
  const [hoveredGroup, setHoveredGroup] = useState<string | null>(null);
  const [clusterMenuOpen, setClusterMenuOpen] = useState(false);

  // 路由变化时，确保当前激活的组被展开（但不收起其他组）
  useEffect(() => {
    const activeGroups = getActiveGroups(pathname);
    setExpandedGroups((prev) => {
      const newGroups = [...prev];
      for (const group of activeGroups) {
        if (!newGroups.includes(group)) {
          newGroups.push(group);
        }
      }
      saveExpandedGroups(newGroups);
      return newGroups;
    });
  }, [pathname]);

  // 展开/收起状态变化时保存到 localStorage
  useEffect(() => {
    saveExpandedGroups(expandedGroups);
  }, [expandedGroups]);

  const isActive = (href: string) => pathname === href;
  const isGroupActive = (group: typeof navGroups[number]) => {
    if (group.href) return pathname === group.href;
    return group.children?.some((child) => pathname === child.href) ?? false;
  };

  const toggleGroup = (key: string) => {
    setExpandedGroups((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]
    );
  };

  return (
    <aside
      className={`h-full flex flex-col relative z-40 overflow-visible ${
        collapsed ? "w-14" : "w-56"
      } ml-4 mr-6 my-6 rounded-2xl bg-[var(--sidebar-bg)] border border-[var(--border-color)]/50 shadow-[0_10px_40px_rgb(0,0,0,0.15),0_0_20px_rgb(0,0,0,0.1)] dark:shadow-[0_10px_40px_rgb(0,0,0,0.5),0_0_20px_rgb(0,0,0,0.3)] ring-1 ring-black/5 dark:ring-white/5`}
      style={{ transition: "width 200ms ease", height: "calc(100% - 48px)" }}
    >
      {/* Logo */}
      <div className={`h-14 flex items-center border-b border-[var(--border-color)]/20 ${collapsed ? "justify-center" : "px-3"}`}>
        <Link href="/about" className="flex-shrink-0">
          {/* 侧栏 32px：用矢量而非位图，小尺寸下边缘更利落 */}
          <img src="/icon.svg" alt="AtlHyper" className="w-8 h-8" />
        </Link>
        {!collapsed && (
          <>
            <span className="flex-1 text-center text-lg font-bold text-primary tracking-tight">AtlHyper</span>
            <a
              href="https://github.com/bukahou/atlhyper"
              target="_blank"
              rel="noopener noreferrer"
              className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-white/10 transition-colors"
              title="GitHub"
            >
              <Github className="w-5 h-5 text-muted hover:text-default" />
            </a>
          </>
        )}
      </div>

      {/* Cluster Selector */}
      <SidebarClusterSelector
        collapsed={collapsed}
        clusterMenuOpen={clusterMenuOpen}
        onSetClusterMenuOpen={setClusterMenuOpen}
      />

      {/* Navigation */}
      <nav
        className={`flex-1 min-h-0 py-3 ${collapsed ? "px-2 overflow-visible" : "px-3 overflow-y-auto"}`}
        style={{ scrollbarWidth: 'thin', scrollbarColor: 'var(--border-color) transparent' }}
      >
        {navGroups.filter((g) => !g.authOnly || isAuthenticated).map((group) => (
          <SidebarNavGroup
            key={group.key}
            group={group}
            collapsed={collapsed}
            isAdmin={isAdmin}
            isActive={isActive}
            isGroupActive={isGroupActive(group)}
            isExpanded={expandedGroups.includes(group.key)}
            hoveredGroup={hoveredGroup}
            onToggleGroup={toggleGroup}
            onHoverGroup={setHoveredGroup}
          />
        ))}
      </nav>

      {/* Footer: User + Settings + Collapse Toggle */}
      <SidebarFooter collapsed={collapsed} onToggle={onToggle} />
    </aside>
  );
}
