/**
 * 国际化类型定义
 */

import type { Language } from "./common";

// 导航菜单翻译
export interface NavTranslations {
  about: string;
  overview: string;
  observe: string;
  slo: string;
  ai: string;
  commands: string;
  cluster: string;
  pod: string;
  node: string;
  deployment: string;
  service: string;
  namespace: string;
  ingress: string;
  event: string;
  metrics: string;
  logs: string;
  admin: string;
  users: string;
  roles: string;
  audit: string;
  clusters: string;
  agents: string;
  settings: string;
  notifications: string;
  aiSettings: string;
  daemonset: string;
  statefulset: string;
  core: string;
  workload: string;
  job: string;
  cronjob: string;
  storage: string;
  pv: string;
  pvc: string;
  policy: string;
  networkPolicy: string;
  resourceQuota: string;
  limitRange: string;
  serviceAccount: string;
  // AIOps
  aiops: string;
  riskDashboard: string;
  incidentsNav: string;
  topologyNav: string;
  // APM
  apm: string;
  // Observe Landing
  // DevOps
  devops: string;
  datasource: string;
  deploy: string;
  github: string;
}

// 通用翻译
export interface CommonTranslations {
  loading: string;
  error: string;
  success: string;
  confirm: string;
  cancel: string;
  save: string;
  delete: string;
  edit: string;
  search: string;
  refresh: string;
  login: string;
  logout: string;
  // 主题
  themeLight: string;
  themeDark: string;
  themeSystem: string;
  // 侧栏
  collapseSidebar: string;
  expandSidebar: string;
  settings: string;
  username: string;
  password: string;
  submit: string;
  noData: string;
  total: string;
  status: string;
  action: string;
  name: string;
  namespace: string;
  createdAt: string;
  close: string;
  view: string;
  details: string;
  filter: string;
  clearAll: string;
  all: string;
  enabled: string;
  disabled: string;
  add: string;
  update: string;
  test: string;
  copy: string;
  copied: string;
  download: string;
  upload: string;
  required: string;
  optional: string;
  yes: string;
  no: string;
  on: string;
  off: string;
  back: string;
  next: string;
  previous: string;
  finish: string;
  reset: string;
  apply: string;
  select: string;
  selected: string;
  none: string;
  more: string;
  less: string;
  expand: string;
  collapse: string;
  show: string;
  hide: string;
  type: string;
  value: string;
  key: string;
  description: string;
  time: string;
  date: string;
  ago: string;
  from: string;
  to: string;
  items: string;
  page: string;
  of: string;
  perPage: string;
  first: string;
  last: string;
  loadMore: string;
  noMore: string;
  retry: string;
  loadFailed: string;
  noCluster: string;
  // OTel 可用性提示
  otelRequired: string;
  otelRequiredDesc: string;
  otelRequiredHint: string;
  // 404 页面
  notFoundTitle: string;
  notFoundDescription: string;
  backToHome: string;
  // K8s 上下文
  k8sContext: string;
  podName: string;
  nodeName: string;
  deploymentName: string;
  namespaceName: string;
  // 演示模式（通用）
  demoMode: string;
  demoModeHint: string;
  demoModeHintAudit: string;
  // 查看权限提示
  viewOnlyHint: string;
}

// 状态翻译
export interface StatusTranslations {
  running: string;
  pending: string;
  failed: string;
  succeeded: string;
  unknown: string;
  ready: string;
  notReady: string;
  healthy: string;
  unhealthy: string;
  degraded: string;
  active: string;
  inactive: string;
  online: string;
  offline: string;
  connected: string;
  disconnected: string;
  terminated: string;
  waiting: string;
  creating: string;
  deleting: string;
  updating: string;
  scaling: string;
  error: string;
  warning: string;
  info: string;
  critical: string;
}

// 审计翻译
export interface AuditTranslations {
  description: string;
  filterLabel: string;
  timeRange: string;
  user: string;
  result: string;
  all: string;
  successOnly: string;
  failedOnly: string;
  total: string;
  successCount: string;
  failedCount: string;
  noRecords: string;
  lastHour: string;
  last24Hours: string;
  last7Days: string;
  allTime: string;
  actions: {
    login: string;
    logout: string;
    podRestart: string;
    podLogs: string;
    nodeCordon: string;
    nodeUncordon: string;
    deploymentScale: string;
    deploymentUpdateImage: string;
    userRegister: string;
    userUpdateRole: string;
    userDelete: string;
    slackConfigUpdate: string;
    unknown: string;
  };
  // 角色标签
  roles: {
    guest: string;
    viewer: string;
    operator: string;
    admin: string;
  };
  // 资源类型标签
  resources: {
    user: string;
    pod: string;
    deployment: string;
    node: string;
    configmap: string;
    secret: string;
    command: string;
    notify: string;
  };
  // 操作+资源组合标签
  actionLabels: {
    loginUser: string;
    executePod: string;
    executeDeployment: string;
    executeNode: string;
    executeCommand: string;
    readPod: string;
    readConfigmap: string;
    readSecret: string;
    createUser: string;
    updateUser: string;
    updateNotify: string;
    deleteUser: string;
  };
  // 操作名称（回退用）
  actionNames: {
    execute: string;
    read: string;
    create: string;
    update: string;
    delete: string;
    login: string;
  };
}

// Pod 页面翻译
export interface PodTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  allNodes: string;
  allStatus: string;
  viewDetails: string;
  viewLogs: string;
  restart: string;
  restartConfirmTitle: string;
  restartConfirmMessage: string;
  containers: string;
  container: string;
  image: string;
  ports: string;
  resources: string;
  limits: string;
  requests: string;
  volumeMounts: string;
  envVars: string;
  conditions: string;
  events: string;
  labels: string;
  annotations: string;
  ownerReferences: string;
  restarts: string;
  age: string;
  node: string;
  ip: string;
  hostIP: string;
  qosClass: string;
  serviceAccount: string;
  phase: string;
  reason: string;
  message: string;
  startTime: string;
  lastState: string;
  currentState: string;
  ready: string;
  started: string;
  restartCount: string;
  logs: string;
  logsTitle: string;
  logsLoading: string;
  logsEmpty: string;
  logsError: string;
  tailLines: string;
  follow: string;
  timestamps: string;
  previous: string;
  selectContainer: string;
  noContainers: string;
  // PodLogsViewer
  showLast: string;
  linesUnit: string;
  searchLogs: string;
  autoScrollToBottom: string;
  autoScroll: string;
  downloadLogs: string;
  fetchLogsFailed: string;
  linesCount: string;
  // Detail modal tabs and sections
  overview: string;
  network: string;
  scheduling: string;
  basicInfo: string;
  resourceUsage: string;
  noVolumes: string;
  lastTerminatedReason: string;
  restartTimes: string;
  controller: string;
}

// Node 页面翻译
export interface NodeTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allStatus: string;
  viewDetails: string;
  cordon: string;
  uncordon: string;
  cordonConfirmTitle: string;
  cordonConfirmMessage: string;
  uncordonConfirmTitle: string;
  uncordonConfirmMessage: string;
  schedulable: string;
  unschedulable: string;
  labels: string;
  annotations: string;
  taints: string;
  conditions: string;
  capacity: string;
  allocatable: string;
  addresses: string;
  nodeInfo: string;
  kubeletVersion: string;
  osImage: string;
  containerRuntime: string;
  kernelVersion: string;
  architecture: string;
  pods: string;
  podsOnNode: string;
  metrics: string;
  cpuUsage: string;
  memoryUsage: string;
  diskUsage: string;
  podCount: string;
  role: string;
  version: string;
  internalIP: string;
  externalIP: string;
  hostname: string;
  // Detail modal
  overview: string;
  resources: string;
  basicInfo: string;
  resourceUsage: string;
  pressureWarning: string;
  memoryPressure: string;
  diskPressure: string;
  pidPressure: string;
  networkUnavailable: string;
  cordoned: string;
  cpuCapacity: string;
  cpuAllocatable: string;
  memoryCapacity: string;
  memoryAllocatable: string;
  podCapacity: string;
  podAllocatable: string;
  ephemeralStorage: string;
  capacityAllocatable: string;
  noConditions: string;
  noTaints: string;
  noLabels: string;
  allArchitectures: string;
  allSchedulable: string;
}

// Deployment 页面翻译
export interface DeploymentTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  allStatus: string;
  viewDetails: string;
  scale: string;
  restart: string;
  updateImage: string;
  scaleConfirmTitle: string;
  scaleConfirmMessage: string;
  restartConfirmTitle: string;
  restartConfirmMessage: string;
  updateImageTitle: string;
  updateImageMessage: string;
  replicas: string;
  readyReplicas: string;
  availableReplicas: string;
  unavailableReplicas: string;
  updatedReplicas: string;
  strategy: string;
  selector: string;
  labels: string;
  annotations: string;
  conditions: string;
  pods: string;
  template: string;
  containers: string;
  currentImage: string;
  newImage: string;
  desiredReplicas: string;
  currentReplicas: string;
  age: string;
  revision: string;
  // Detail modal tabs and sections
  overview: string;
  scheduling: string;
  replicaSets: string;
  basicInfo: string;
  adjustReplicas: string;
  updateStrategy: string;
  otherConfig: string;
  schedulingConfig: string;
  desired: string;
  ready: string;
  available: string;
  updated: string;
  paused: string;
  current: string;
  noContainers: string;
  noReplicaSets: string;
  noLabels: string;
  noAnnotations: string;
  image: string;
  ports: string;
  probes: string;
  envVars: string;
  volumeMounts: string;
  loadFailed: string;
  confirmUpdateImage: string;
  confirmUpdateImageMessage: string;
  confirmScale: string;
  confirmScaleMessage: string;
  hostNetwork: string;
  strategyType: string;
}

// Service 页面翻译
export interface ServiceTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  allTypes: string;
  viewDetails: string;
  serviceType: string;
  clusterIP: string;
  externalIP: string;
  ports: string;
  port: string;
  targetPort: string;
  nodePort: string;
  protocol: string;
  selector: string;
  labels: string;
  annotations: string;
  endpoints: string;
  noEndpoints: string;
  sessionAffinity: string;
  externalTrafficPolicy: string;
  loadBalancerIP: string;
  age: string;
  // Detail modal tabs and sections
  overview: string;
  basicInfo: string;
  loadFailed: string;
  noPorts: string;
  noSelector: string;
  endpointStatus: string;
  ready: string;
  notReady: string;
  total: string;
  trafficPolicy: string;
}

// Namespace 页面翻译
export interface NamespaceTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allStatus: string;
  viewDetails: string;
  labels: string;
  annotations: string;
  resourceQuotas: string;
  limitRanges: string;
  configMaps: string;
  secrets: string;
  pods: string;
  services: string;
  deployments: string;
  age: string;
  phase: string;
  finalizers: string;
  noConfigMaps: string;
  noSecrets: string;
  viewData: string;
  dataTitle: string;
  entries: string;
  secretType: string;
  masked: string;
  reveal: string;
  // Detail modal tabs and sections
  overview: string;
  quotas: string;
  basicInfo: string;
  loadFailed: string;
  podStatus: string;
  total: string;
  workloads: string;
  network: string;
  config: string;
  resourceUsage: string;
  utilization: string;
  noQuotas: string;
  noLabels: string;
  noAnnotations: string;
  noData: string;
}

// Ingress 页面翻译
export interface IngressTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  viewDetails: string;
  ingressClass: string;
  rules: string;
  host: string;
  path: string;
  pathType: string;
  backend: string;
  serviceName: string;
  servicePort: string;
  tls: string;
  tlsHosts: string;
  secretName: string;
  labels: string;
  annotations: string;
  defaultBackend: string;
  noRules: string;
  age: string;
  // Detail modal
  overview: string;
  routingRules: string;
  basicInfo: string;
  loadFailed: string;
  tlsStatus: string;
  tlsEnabled: string;
  tlsDisabled: string;
  ruleStatistics: string;
  pathCount: string;
  tlsCertificates: string;
  noTlsConfig: string;
  noAnnotations: string;
}

// Alert 页面翻译
export interface AlertTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allSeverities: string;
  allTypes: string;
  viewDetails: string;
  severity: string;
  type: string;
  source: string;
  message: string;
  timestamp: string;
  acknowledged: string;
  acknowledge: string;
  resolve: string;
  silence: string;
  labels: string;
  annotations: string;
  noAlerts: string;
  critical: string;
  warning: string;
  info: string;
  // 告警列表页面
  loadFailed: string;
  refresh: string;
  aiAnalyze: string;
  noAlertsTitle: string;
  noAlertsDescription: string;
  time: string;
  level: string;
  resourceType: string;
  namespace: string;
  resourceName: string;
  reason: string;
  totalAlerts: string;
  selectedCount: string;
  selectAll: string;
  analyzeHint: string;
}

// Overview 页面翻译
export interface OverviewTranslations {
  clusterHealth: string;
  nodeReady: string;
  podHealthy: string;
  nodes: string;
  clusterAvgCpu: string;
  clusterAvgMem: string;
  alerts: string;
  nodeResourceUsage: string;
  noNodeData: string;
  recentAlerts: string;
  noRecentAlerts: string;
  workloadSummary: string;
  podStatus: string;
  cpuPeak: string;
  memPeak: string;
  alertDetails: string;
  time: string;
  kind: string;
  reason: string;
  deploymentsLabel: string;
  daemonSetsLabel: string;
  statefulSetsLabel: string;
  jobsLabel: string;
  run: string;
  done: string;
  fail: string;
  // SLO 概览卡片
  sloOverview: string;
  viewSloDetail: string;
}

// Workbench 页面翻译
export interface WorkbenchTranslations {
  pageDescription: string;
  recentEvents: string;
  quickActions: string;
  noEvents: string;
  eventType: string;
  eventMessage: string;
  eventTime: string;
  podOperations: string;
  nodeOperations: string;
  deploymentOperations: string;
  // 快捷入口描述
  enter: string;
  aiDescription: string;
  clustersDescription: string;
  agentsDescription: string;
  notificationsDescription: string;
  // 开发中
  comingSoon: string;
  comingSoonDesc: string;
}

// Users 页面翻译
export interface UsersTranslations {
  pageDescription: string;
  addUser: string;
  editUser: string;
  deleteUser: string;
  deleteConfirmTitle: string;
  deleteConfirmMessage: string;
  username: string;
  displayName: string;
  email: string;
  role: string;
  status: string;
  lastLogin: string;
  lastLoginIP: string;
  createdAt: string;
  roleAdmin: string;
  roleOperator: string;
  roleViewer: string;
  statusActive: string;
  statusDisabled: string;
  passwordPlaceholder: string;
  changeRole: string;
  changeStatus: string;
  enable: string;
  disable: string;
  cannotDeleteAdmin: string;
  cannotDisableAdmin: string;
}

// Roles 页面翻译
export interface RolesTranslations {
  pageDescription: string;
  // 角色描述
  adminDescription: string;
  operatorDescription: string;
  viewerDescription: string;
  // 权限矩阵
  permissionMatrix: string;
  permissionMatrixDescription: string;
  resource: string;
  notes: string;
  // 分类
  categorySystem: string;
  categoryCluster: string;
  categoryMonitoring: string;
  categoryAI: string;
  // 资源
  userManagement: string;
  roleAssignment: string;
  auditLogs: string;
  notificationConfig: string;
  aiConfig: string;
  metricsView: string;
  logsView: string;
  alertRules: string;
  aiDiagnosis: string;
  aiWorkbench: string;
  // 权限标签
  permissionFull: string;
  permissionReadOnly: string;
  permissionPartial: string;
  permissionNone: string;
  // 权限说明
  permissionLevelDescription: string;
  fullPermission: string;
  fullPermissionDesc: string;
  readOnlyPermission: string;
  readOnlyPermissionDesc: string;
  partialPermission: string;
  partialPermissionDesc: string;
  noPermission: string;
  noPermissionDesc: string;
  // 备注
  noteViewUserList: string;
  noteOperatorSilenceAlert: string;
}

// Clusters 页面翻译
export interface ClustersTranslations {
  pageDescription: string;
  addCluster: string;
  clusterName: string;
  clusterID: string;
  status: string;
  nodeCount: string;
  podCount: string;
  version: string;
  lastSync: string;
  connected: string;
  disconnected: string;
  noClusters: string;
}

// Agents 页面翻译
export interface AgentsTranslations {
  pageDescription: string;
  agentName: string;
  clusterID: string;
  status: string;
  version: string;
  lastHeartbeat: string;
  uptime: string;
  connected: string;
  disconnected: string;
  noAgents: string;
}

// Notifications 页面翻译
export interface NotificationsTranslations {
  pageDescription: string;
  loadFailed: string;
  // Slack
  slackNotify: string;
  slackConfig: string;
  webhookUrl: string;
  webhookUrlHint: string;
  channel: string;
  // Email
  emailNotify: string;
  smtpServer: string;
  smtpServerPlaceholder: string;
  smtpCustom: string;
  smtpCustomPlaceholder: string;
  port: string;
  portHint: string;
  emailAccount: string;
  emailAccountHint: string;
  password: string;
  passwordPlaceholder: string;
  tlsEncryption: string;
  tlsHint: string;
  recipients: string;
  recipientsPlaceholder: string;
  // 状态
  statusEnabled: string;
  statusIncomplete: string;
  statusDisabled: string;
  configIncomplete: string;
  // 操作
  enabled: string;
  test: string;
  save: string;
  testMessage: string;
  testSuccess: string;
  testFailed: string;
  saveSuccess: string;
  saveFailed: string;
  slackSaved: string;
  emailSaved: string;
  // SMTP 预设标签
  smtpPresets: {
    qq: string;
    netease163: string;
    netease126: string;
    tencentEnterprise: string;
    aliEnterprise: string;
    feishu: string;
  };
  // TagInput 消息
  tagInputPlaceholder: string;
  tagInputDuplicate: string;
  tagInputInvalidFormat: string;
}

// Login 页面翻译
export interface LoginTranslations {
  title: string;
  subtitle: string;
  usernamePlaceholder: string;
  passwordPlaceholder: string;
  loginButton: string;
  loggingIn: string;
  loginSuccess: string;
  loginFailed: string;
  invalidCredentials: string;
  sessionExpired: string;
  pleaseLogin: string;
}

// Confirm Dialog 翻译
export interface ConfirmTranslations {
  defaultTitle: string;
  defaultMessage: string;
  confirmButton: string;
  cancelButton: string;
  deleteTitle: string;
  deleteMessage: string;
  warningTitle: string;
  dangerTitle: string;
}

// Table 翻译
export interface TableTranslations {
  noData: string;
  loading: string;
  error: string;
  showing: string;
  entries: string;
  pageOf: string;
  rowsPerPage: string;
  firstPage: string;
  previousPage: string;
  nextPage: string;
  lastPage: string;
  sortAsc: string;
  sortDesc: string;
  filterPlaceholder: string;
}

// DaemonSet 翻译
export interface DaemonSetTranslations {
  pageDescription: string;
  desiredScheduled: string;
  currentScheduled: string;
  numberReady: string;
  numberAvailable: string;
  numberMisscheduled: string;
  updateStrategy: string;
  selector: string;
  labels: string;
  annotations: string;
  conditions: string;
  pods: string;
  template: string;
  containers: string;
  nodeSelector: string;
  tolerations: string;
  age: string;
  // Detail modal tabs and sections
  overview: string;
  strategy: string;
  basicInfo: string;
  desired: string;
  ready: string;
  available: string;
  current: string;
  updated: string;
  noContainers: string;
  noLabels: string;
  noAnnotations: string;
  image: string;
  ports: string;
  loadFailed: string;
  nodeScheduleStatus: string;
  abnormalStatus: string;
  unavailable: string;
  misscheduled: string;
  strategyType: string;
  otherConfig: string;
  scheduling: string;
  moreItems: string;
}

// StatefulSet 翻译
export interface StatefulSetTranslations {
  pageDescription: string;
  replicas: string;
  readyReplicas: string;
  currentReplicas: string;
  updatedReplicas: string;
  updateStrategy: string;
  serviceName: string;
  podManagementPolicy: string;
  selector: string;
  labels: string;
  annotations: string;
  conditions: string;
  volumeClaimTemplates: string;
  pods: string;
  template: string;
  containers: string;
  age: string;
  revision: string;
  // Detail modal tabs and sections
  overview: string;
  strategy: string;
  storage: string;
  basicInfo: string;
  desired: string;
  ready: string;
  available: string;
  current: string;
  updated: string;
  noContainers: string;
  noLabels: string;
  noAnnotations: string;
  image: string;
  ports: string;
  loadFailed: string;
  replicaStatus: string;
  currentRevision: string;
  updateRevision: string;
  strategyType: string;
  podManagement: string;
  pvcRetentionPolicy: string;
  whenDeleted: string;
  whenScaled: string;
  scheduling: string;
  moreItems: string;
  noPvcTemplates: string;
}

// Placeholder 页面翻译
export interface PlaceholderTranslations {
  developingTitle: string;
  developingMessage: string;
}

// Job 页面翻译
export interface JobTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  active: string;
  succeeded: string;
  failed: string;
  completions: string;
  parallelism: string;
  duration: string;
  age: string;
  statusComplete: string;
  statusRunning: string;
  statusFailed: string;
  detailOverview: string;
  detailLabels: string;
  detailTimeInfo: string;
  detailStartTime: string;
  detailFinishTime: string;
  detailBasicInfo: string;
  detailOwner: string;
  detailCreatedAt: string;
  detailNoLabels: string;
  detailContainers: string;
  detailNoContainers: string;
  detailImage: string;
  detailPorts: string;
  detailProbes: string;
  detailConfig: string;
  detailCompletions: string;
  detailParallelism: string;
  detailBackoffLimit: string;
  detailConditions: string;
  detailNoAnnotations: string;
}

// CronJob 页面翻译
export interface CronJobTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  schedule: string;
  suspend: string;
  suspended: string;
  activeJobs: string;
  lastSchedule: string;
  age: string;
  detailOverview: string;
  detailLabels: string;
  detailScheduleInfo: string;
  detailLastSuccess: string;
  detailBasicInfo: string;
  detailOwner: string;
  detailCreatedAt: string;
  detailNoLabels: string;
  detailContainers: string;
  detailNoContainers: string;
  detailImage: string;
  detailPorts: string;
  detailProbes: string;
  detailConfig: string;
  detailConcurrencyPolicy: string;
  detailSuccessHistoryLimit: string;
  detailFailedHistoryLimit: string;
  detailNoAnnotations: string;
}

// Storage (PV/PVC) 页面翻译
export interface StorageTranslations {
  pvDescription: string;
  pvcDescription: string;
  searchPlaceholder: string;
  capacity: string;
  phase: string;
  storageClass: string;
  reclaimPolicy: string;
  accessModes: string;
  volumeName: string;
  requested: string;
  bound: string;
  available: string;
  released: string;
  pending: string;
  lost: string;
  age: string;
  // 详情
  detailOverview: string;
  detailLabels: string;
  detailStorageInfo: string;
  detailBasicInfo: string;
  detailCreatedAt: string;
  detailNoLabels: string;
  detailVolumeSourceType: string;
  detailClaimRef: string;
  detailVolumeMode: string;
  detailNoAnnotations: string;
}

// Policy (NetworkPolicy, ResourceQuota, LimitRange, ServiceAccount) 页面翻译
export interface PolicyTranslations {
  networkPolicyDescription: string;
  resourceQuotaDescription: string;
  limitRangeDescription: string;
  serviceAccountDescription: string;
  searchPlaceholder: string;
  policyTypes: string;
  ingressRules: string;
  egressRules: string;
  hardUsed: string;
  resourceTypes: string;
  itemsCount: string;
  types: string;
  secrets: string;
  imagePullSecrets: string;
  automountToken: string;
  withSecrets: string;
  rulesCount: string;
  age: string;
  // 详情
  detailBasicInfo: string;
  detailCreatedAt: string;
  detailRuleStats: string;
  detailPodSelector: string;
  detailQuotaUsage: string;
  detailNoQuota: string;
  detailScopes: string;
  detailLimitEntries: string;
  detailNoEntries: string;
  detailSecretStats: string;
  detailOverview: string;
  detailLabels: string;
  detailRules: string;
  detailNoLabels: string;
  detailNoAnnotations: string;
  detailPeers: string;
  detailPorts: string;
  detailNoRules: string;
  detailSecretNames: string;
  detailImagePullSecretNames: string;
  detailNoSecrets: string;
}

// SLO 页面翻译
export interface SLOTranslations extends TimeRangePickerTranslations {
  windowDegraded: string;
  windowDegradedHint: string;
  // 页面标题
  pageTitle: string;
  pageDescription: string;
  noData: string;
  noDataHint: string;
  refreshing: string;
  lastUpdated: string;
  loadFailed: string;
  noCluster: string;
  // 状态
  healthy: string;
  warning: string;
  critical: string;
  degraded: string;
  atRisk: string;
  breached: string;
  unknown: string;
  // 趋势
  trendUp: string;
  trendDown: string;
  trendStable: string;
  // 时间范围
  day: string;
  week: string;
  month: string;
  selectPeriod: string;
  sloWindow: string;
  burnRate: string;
  burnRateHint: string;
  window: string;
  rate: string;
  eventCount: string;
  badEvents: string;
  allowedEvents: string;
  overspent: string;
  currentSli: string;
  goodBad: string;
  exhaustIn: string;
  hours: string;
  viewTraces: string;
  viewErrorLogs: string;
  sloListTitle: string;
  sloListHint: string;
  sloWindowHint: string;
  days: string;
  previousPeriod: string;
  currentVsPrevious: string;
  // 指标
  availability: string;
  p95Latency: string;
  p99Latency: string;
  latency: string;
  errorRate: string;
  rps: string;
  throughput: string;
  totalRequests: string;
  successRequests: string;
  errorRequests: string;
  avgThroughput: string;
  // 详情弹窗
  domainDetail: string;
  targetDomain: string;
  currentMetrics: string;
  sloTarget: string;
  sloStatus: string;
  sloAchievement: string;
  errorBudget: string;
  errorBudgetDetail: string;
  remainingBudget: string;
  allowedErrors: string;
  actualErrors: string;
  remainingQuota: string;
  trafficStats: string;
  history: string;
  noTarget: string;
  setTarget: string;
  editTarget: string;
  deleteTarget: string;
  configTarget: string;
  configSloTarget: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  remaining: string;
  consumed: string;
  actual: string;
  target: string;
  threshold: string;
  achieved: string;
  notAchieved: string;
  exceeded: string;
  // Tabs
  tabSloStatus: string;
  tabServices: string;
  tabCompare: string;
  tabOverview: string;
  tabLatency: string;
  // 服务网格
  serviceTopology: string;
  mtls: string;
  inbound: string;
  outbound: string;
  noCallData: string;
  callRelation: string;
  p50Latency: string;
  avgLatency: string;
  // 趋势图
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
  // 服务相关
  services: string;
  servicesCount: string;
  backendServices: string;
  noServiceData: string;
  totalBackendServices: string;
  // 汇总卡片
  totalServices: string;
  ingressBackends: string;
  monitoredDomains: string;
  avgAvailability: string;
  avgP95: string;
  totalRPS: string;
  alertCount: string;
  severe: string;
  domainSloStatus: string;
  // 图表
  hourlyTrend: string;
  dailyTrend: string;
  // 单位
  ms: string;
  percent: string;
  reqPerSec: string;
  // 操作
  viewDetail: string;
  save: string;
  saving: string;
  cancel: string;
  saveFailed: string;
  // 延迟分布 / 服务详情
  latencyDistribution: string;
  methodBreakdown: string;
  statusCodeBreakdown: string;
  clearSelection: string;
  requests: string;
  loading: string;
  // 说明
  dataSourceTitle: string;
  dataSourceDesc: string;
}

// Commands 页面翻译
export interface CommandsTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allSources: string;
  allStatus: string;
  allActions: string;
  source: string;
  target: string;
  params: string;
  result: string;
  duration: string;
  noCommands: string;
  viewDetails: string;
  commandId: string;
  errorMessage: string;
  createdAt: string;
  startedAt: string;
  finishedAt: string;
  sources: {
    web: string;
    ai: string;
  };
  statuses: {
    pending: string;
    running: string;
    success: string;
    failed: string;
    timeout: string;
  };
  actions: {
    restart: string;
    scale: string;
    delete_pod: string;
    cordon: string;
    uncordon: string;
    update_image: string;
  };
}

// AI Settings 页面翻译
export interface AISettingsPageTranslations {
  // 页面标题
  pageTitle: string;
  pageDescription: string;
  // 全局设置
  globalSettings: string;
  globalSettingsDesc: string;
  toolTimeout: string;
  toolTimeoutDesc: string;
  seconds: string;
  saveTimeout: string;
  // 提供商列表
  providerList: string;
  addProvider: string;
  addFirstProvider: string;
  noProviders: string;
  // 提供商卡片
  settings: string;
  edit: string;
  delete: string;
  model: string;
  apiKeyConfigured: string;
  // 提供商弹窗
  newProvider: string;
  editProvider: string;
  addProviderTitle: string;
  editProviderTitle: string;
  name: string;
  namePlaceholder: string;
  providerName: string;
  providerNamePlaceholder: string;
  provider: string;
  providerType: string;
  selectProvider: string;
  apiKey: string;
  apiKeyPlaceholder: string;
  apiKeyUpdatePlaceholder: string;
  apiKeyHint: string;
  current: string;
  selectModel: string;
  customModel: string;
  customModelPlaceholder: string;
  description: string;
  descriptionPlaceholder: string;
  cancel: string;
  save: string;
  saving: string;
  // Toast 消息
  loadFailed: string;
  requiredFields: string;
  apiKeyRequired: string;
  providerUpdated: string;
  providerAdded: string;
  saveFailed: string;
  confirmDelete: string;
  providerDeleted: string;
  deleteFailed: string;
  timeoutSaved: string;
  // 角色总览
  roleOverview: string;
  roleBackground: string;
  roleChat: string;
  roleAnalysis: string;
  roleUnassigned: string;
  // Base URL
  baseUrl: string;
  baseUrlPlaceholder: string;
  baseUrlHint: string;
  // 角色标签
  rolesUpdated: string;
  // 预算配置
  budgetConfig: string;
  budgetConfigDesc: string;
  // 日限额
  dailyInputTokenLimit: string;
  dailyOutputTokenLimit: string;
  dailyCallLimit: string;
  dailyInputTokens: string;
  dailyOutputTokens: string;
  dailyCalls: string;
  // 月限额
  monthlyInputTokenLimit: string;
  monthlyOutputTokenLimit: string;
  monthlyCallLimit: string;
  monthlyInputTokens: string;
  monthlyOutputTokens: string;
  monthlyCalls: string;
  autoTriggerMinSeverity: string;
  autoTriggerMinSeverityDesc: string;
  autoTriggerMode: string;
  autoTriggerModeDesc: string;
  triggerModeAuto: string;
  triggerModeManual: string;
  triggerModeSchedule: string;
  scheduleWindow: string;
  scheduleWindowDesc: string;
  severityCritical: string;
  severityHigh: string;
  severityMedium: string;
  severityLow: string;
  severityOff: string;
  budgetSaved: string;
  budgetSaveFailed: string;
  unlimited: string;
  // 角色互斥
  roleAssignConflict: string;
  roleAssignSuccess: string;
  // 降级 Provider
  fallbackProvider: string;
  noFallback: string;
  // 调用历史
  usageHistory: string;
  usageHistoryDesc: string;
  allRoles: string;
  noReports: string;
  trigger: string;
  inputTokens: string;
  outputTokens: string;
  duration: string;
  loadMore: string;
}

// AI Chat 页面翻译
export interface AIChatPageTranslations {
  // 状态页面
  loading: string;
  checkingConfig: string;
  notConfigured: string;
  notConfiguredDesc: string;
  chatNotAssigned: string;
  chatNotAssignedDesc: string;
  goToSettings: string;
  // 演示模式
  demoMode: string;
  demoModeDesc: string;
  login: string;
  // 告警分析
  alertAnalysis: string;
  // ChatPanel
  conversationHistory: string;
  executionDetails: string;
  aiAssistant: string;
  aiAssistantDesc: string;
  // 快捷问题
  quickQuestions: {
    podRestart: string;
    nodeResource: string;
    alertEvents: string;
    deploymentStatus: string;
  };
  // ChatInput
  chatInput: {
    placeholderStreaming: string;
    placeholderNormal: string;
    stopButton: string;
    sendButton: string;
    disclaimer: string;
  };
  // InspectorPanel
  inspector: {
    title: string;
    clusterContext: string;
    clusterId: string;
    notSelected: string;
    status: string;
    querying: string;
    connected: string;
    currentQuestion: string;
    thinkingRounds: string;
    inProgress: string;
    executedCommands: string;
    commandsUnit: string;
    roundsUnit: string;
    inputTokens: string;
    outputTokens: string;
    totalTokens: string;
    waitingQuestion: string;
    conversationOverview: string;
    conversationRounds: string;
    totalCommands: string;
    totalTokensLabel: string;
    aiCapabilities: string;
    canDo: string;
    cannotDo: string;
    capQueryResources: string;
    capViewLogs: string;
    capViewEvents: string;
    capFilterByLabel: string;
    capNoWrite: string;
    capNoSecrets: string;
    capNoSystemNs: string;
    capNoSensitive: string;
  };
  // Demo mock 数据
  demo: {
    conv1Title: string;
    conv2Title: string;
    msg1User: string;
    msg2Assistant: string;
    msg3User: string;
    msg4Assistant: string;
  };
  // ConversationSidebar
  sidebar: {
    title: string;
    newConversation: string;
    noConversations: string;
    deleteConversation: string;
    today: string;
    yesterday: string;
    past7Days: string;
    past30Days: string;
    older: string;
  };
  // ExecutionBlock
  execution: {
    roundLabel: string;
    executing: string;
    commandsUnit: string;
    thinkingRounds: string;
    commandsCount: string;
    completed: string;
    hasFailed: string;
    success: string;
    failed: string;
  };
}

// About 介绍页翻译
export interface AboutTranslations {
  // Hero
  subtitle: string;
  description: string;
  heroStatComponents: string;
  heroStatLayers: string;
  heroStatLanguages: string;
  heroStatLicense: string;
  // 状态徽章
  statusDone: string;
  statusPlanned: string;
  // 五层可观测性架构
  sectionArchitecture: string;
  sectionArchitectureDesc: string;
  layer1Title: string;
  layer1Desc: string;
  layer1Source: string;
  layer1Metrics: string;
  layer2Title: string;
  layer2Desc: string;
  layer2Source: string;
  layer2Metrics: string;
  layer3Title: string;
  layer3Desc: string;
  layer3Source: string;
  layer3Metrics: string;
  layer4Title: string;
  layer4Desc: string;
  layer4Source: string;
  layer4Metrics: string;
  layer5Title: string;
  layer5Desc: string;
  layer5Source: string;
  layer5Metrics: string;
  // Drill-down 层间关联
  drilldownLabel: string;
  drilldown12: string;
  drilldown23: string;
  drilldown34: string;
  drilldown45: string;
  // AI 贯穿
  aiTitle: string;
  aiDesc: string;
  aiScenarioTitle: string;
  aiScenarioStep1: string;
  aiScenarioStep2: string;
  aiScenarioStep3: string;
  aiScenarioStep4: string;
  aiScenarioStep5: string;
  aiFullStack: string;
  // 功能模块
  sectionFeatures: string;
  featureClusterTitle: string;
  featureClusterDesc: string;
  featureSloTitle: string;
  featureSloDesc: string;
  featureTopologyTitle: string;
  featureTopologyDesc: string;
  featureAiTitle: string;
  featureAiDesc: string;
  featureCommandTitle: string;
  featureCommandDesc: string;
  featureAlertTitle: string;
  featureAlertDesc: string;
  featureMetricsTitle: string;
  featureMetricsDesc: string;
  featureApmTitle: string;
  featureApmDesc: string;
  featureLogsTitle: string;
  featureLogsDesc: string;
  // 技术栈
  sectionTechStack: string;
  techMasterTitle: string;
  techMasterStack: string;
  techMasterDesc: string;
  techAgentTitle: string;
  techAgentStack: string;
  techAgentDesc: string;
  techOtelTitle: string;
  techOtelStack: string;
  techOtelDesc: string;
  techWebTitle: string;
  techWebStack: string;
  techWebDesc: string;
  // AIOps 引擎
  aiopsEngineTitle: string;
  aiopsEngineDesc: string;
  aiopsStatusPartial: string;
  aiopsDepGraphTitle: string;
  aiopsDepGraphDesc: string;
  aiopsBaselineTitle: string;
  aiopsBaselineDesc: string;
  aiopsRiskTitle: string;
  aiopsRiskDesc: string;
  aiopsStateMachineTitle: string;
  aiopsStateMachineDesc: string;
  aiopsFlow1: string;
  aiopsFlow2: string;
  aiopsFlow3: string;
  aiopsFlow4: string;
  aiopsVision: string;
  // 开源信息
  sectionOpenSource: string;
  openSourceDesc: string;
  openSourceRequirements: string;
  // 层级详情弹窗
  detailSectionWhat: string;
  detailSectionRole: string;
  detailSectionTools: string;
  detailSectionAtlhyper: string;
  detailClickHint: string;
  layer1DetailWhat: string;
  layer1DetailRole: string;
  layer1DetailTools: string;
  layer1DetailAtlhyper: string;
  layer2DetailWhat: string;
  layer2DetailRole: string;
  layer2DetailTools: string;
  layer2DetailAtlhyper: string;
  layer3DetailWhat: string;
  layer3DetailRole: string;
  layer3DetailTools: string;
  layer3DetailAtlhyper: string;
  layer4DetailWhat: string;
  layer4DetailRole: string;
  layer4DetailTools: string;
  layer4DetailAtlhyper: string;
  layer5DetailWhat: string;
  layer5DetailRole: string;
  layer5DetailTools: string;
  layer5DetailAtlhyper: string;
}

// 节点指标翻译
interface NodeMetricsTranslations extends TimeRangePickerTranslations {
  rangeAffectsTrendOnly: string;
  // 页面级
  pageDescription: string;
  lastUpdate: string;
  retry: string;
  noCluster: string;
  noClusterDesc: string;
  noMetricsData: string;
  noMetricsDesc: string;
  loadFailed: string;
  allNodes: string;
  // 汇总
  summary: {
    nodes: string;
    warnings: string;
    avgCpu: string;
    avgMemory: string;
    maxTemp: string;
    avgDisk: string;
    nodesNeedAttention: string;
  };
  // 节点
  node: {
    controlPlane: string;
    worker: string;
    uptime: string;
  };
  // CPU
  cpu: {
    title: string;
    usage: string;
    coreUsage: string;
    threads: string;
    load: string;
    saturation: string;
    perCore: string;
  };
  // Memory
  memory: {
    title: string;
    total: string;
    usage: string;
    used: string;
    available: string;
    cached: string;
    buffers: string;
    swap: string;
    swapUsed: string;
    noSwap: string;
    oomKill: string;
    oomKillNone: string;
  };
  // Disk
  disk: {
    title: string;
    mounts: string;
    read: string;
    write: string;
    iops: string;
    ioUtil: string;
    totalIops: string;
    inode: string;
    await: string;
    queue: string;
    readOnly: string;
    utilization: string;
    saturation: string;
    errors: string;
    healthy: string;
  };
  // Network
  network: {
    title: string;
    interfaces: string;
    physicalInterfaces: string;
    virtualInterfaces: string;
    showVirtual: string;
    hideVirtual: string;
    receive: string;
    transmit: string;
    rxShort: string;
    txShort: string;
    rxPackets: string;
    txPackets: string;
    errors: string;
    dropped: string;
    cumulativeStats: string;
    sinceBootup: string;
  };
  // PSI
  psi: {
    title: string;
    description: string;
    someDesc: string;
    fullDesc: string;
  };
  // TCP
  tcp: {
    title: string;
    activeConnections: string;
    alloc: string;
    inUse: string;
    socketsUsed: string;
    softnetDropped: string;
    softnetSqueezed: string;
  };
  // VMStat
  vmstat: {
    title: string;
    description: string;
    pageFaults: string;
    majorFaults: string;
    perSecond: string;
    perSecondDisk: string;
    swapIn: string;
    swapOut: string;
    pagesPerSec: string;
    highMajorFaults: string;
    activeSwapIO: string;
  };
  // System Resources
  system: {
    title: string;
    runQueue: string;
    blocked: string;
    arpEntries: string;
    fileDescriptors: string;
    conntrackTable: string;
    used: string;
    entropyPool: string;
    bits: string;
    entropySufficient: string;
    entropyLow: string;
    ntpSync: string;
    synced: string;
    notSynced: string;
    offset: string;
  };
  // Temperature
  temperature: {
    title: string;
    cpuTemp: string;
    sensors: string;
    critical: string;
    highWarning: string;
    na: string;
    groupCpu: string;
    groupDisk: string;
    groupOther: string;
    power: string;
    fanSpeed: string;
  };
  // Compare（节点对比表）
  compare: {
    title: string;
    subtitle: string;
    node: string;
    overall: string;
    noData: string;
    normal: string;
    alarm: string;
    columns: {
      cpuTemp: string;
      diskTemp: string;
      undervolt: string;
      freqRatio: string;
      diskAwait: string;
      diskUsage: string;
      cpu: string;
      psiCpu: string;
      mem: string;
      psiMem: string;
      netErr: string;
    };
  };
  // Hardware（硬件健康矩阵）
  hardware: {
    title: string;
    subtitle: string;
    node: string;
    cpuUsage: string;
    memUsage: string;
    diskUsage: string;
    cpuTemp: string;
    diskTemp: string;
    otherTemp: string;
    undervolt: string;
    fan: string;
    cpuFreq: string;
    diskAwait: string;
    overall: string;
    noData: string;
    normal: string;
    alarm: string;
    stopped: string;
    threshold: string;
    tiles: {
      maxTemp: string;
      maxDiskTemp: string;
      undervoltNodes: string;
      throttledNodes: string;
      maxDiskAwait: string;
      nodes: string;
      none: string;
    };
    status: {
      good: string;
      warn: string;
      crit: string;
    };
  };
  // Chart
  chart: {
    title: string;
    noData: string;
  };
  // Cluster Overview Chart
  overview: {
    title: string;
    top5: string;
    noData: string;
  };
}

// AIOps 翻译
export interface AIOpsTranslations {
  // 页面标题
  riskDashboard: string;
  incidents: string;
  topology: string;
  pageDescription: string;

  // 风险等级
  riskLevel: {
    healthy: string;
    low: string;
    medium: string;
    high: string;
    critical: string;
  };

  // 风险仪表盘
  clusterRisk: string;
  riskScore: string;
  riskTrend: string;
  topRiskEntities: string;
  anomalyCount: string;
  totalEntities: string;

  // 实体风险
  entityKey: string;
  entityType: string;
  rLocal: string;
  rFinal: string;
  riskLevelLabel: string;
  firstAnomaly: string;
  noAnomaly: string;
  noAnomalyDetail: string;
  depTrace: string;
  depUpstream: string;
  depDownstream: string;
  depNone: string;
  patternTitle: string;
  patternRecur: string;
  patternAvgDuration: string;
  patternLast: string;
  patternFirst: string;
  metricName: string;
  currentValue: string;
  baseline: string;
  baselineTitle: string;
  baselineDesc: string;
  baselineLearning: string;
  baselineReady: string;
  baselineNormal: string;
  baselineFluctuation: string;
  baselineSamples: string;
  baselineStable: string;
  baselineUnavailable: string;
  baselineLoadFailed: string;
  deviation: string;
  isAnomaly: string;

  // 事件
  incidentId: string;
  incidentState: string;
  severity: string;
  rootCause: string;
  peakRisk: string;
  startedAt: string;
  resolvedAt: string;
  duration: string;
  recurrence: string;
  affectedEntities: string;
  timeline: string;
  role: string;

  // 事件状态
  state: {
    warning: string;
    incident: string;
    recovery: string;
    stable: string;
  };
  stateDesc: {
    warning: string;
    incident: string;
    recovery: string;
    stable: string;
  };

  // 严重度
  severityLevel: {
    low: string;
    medium: string;
    high: string;
    critical: string;
  };
  severityDesc: {
    low: string;
    medium: string;
    high: string;
    critical: string;
  };
  incidentStateTooltip: string;
  severityTooltip: string;

  // 统计
  stats: {
    total: string;
    active: string;
    mttr: string;
    recurrenceRate: string;
    bySeverity: string;
    topRootCauses: string;
  };

  // 时间线事件类型
  timelineEvent: {
    anomaly_detected: string;
    state_change: string;
    metric_spike: string;
    root_cause_identified: string;
    recovery_started: string;
    recurrence: string;
  };

  // 拓扑图
  dependencyGraph: string;
  nodeDetail: string;
  upstream: string;
  downstream: string;
  causalChain: string;
  causalTree: string;
  propagation: string;
  selectNode: string;
  viewService: string;
  viewAnomaly: string;
  viewFull: string;
  noAnomalies: string;
  // topology legend
  nodeTypeService: string;
  nodeTypeIngress: string;
  nodeTypePod: string;
  nodeTypeNode: string;
  legendHealthy: string;
  legendWarning: string;
  legendCritical: string;
  pauseAutoRefresh: string;
  resumeAutoRefresh: string;

  // 通用
  noData: string;
  loading: string;
  autoRefresh: string;
  lastUpdate: string;
  noCluster: string;
  noClusterDesc: string;
  loadFailed: string;
  retry: string;
  minutesAgo: string;
  hoursAgo: string;
  justNow: string;
  minutes: string;

  // AI 增强
  ai: {
    analyze: string;
    analyzing: string;
    analysisFailed: string;
    summary: string;
    rootCauseAnalysis: string;
    recommendations: string;
    similarIncidents: string;
    priority: string;
    action: string;
    reason: string;
    impact: string;
    similarity: string;
    noSimilar: string;
    generatedAt: string;
    regenerate: string;
    // AI 报告
    aiReports: string;
    noReports: string;
    generateAnalysis: string;
    reanalyze: string;
    triggerDeepAnalysis: string;
    analysisSubmitted: string;
    analysisInProgress: string;
    fromBackground: string;
    fromAnalysis: string;
    fromManual: string;
    triggerIncidentCreated: string;
    triggerStateChanged: string;
    triggerManual: string;
    investigationSteps: string;
    evidenceChain: string;
    round: string;
    toolCalls: string;
    tokens: string;
    duration: string;
    expandSteps: string;
    collapseSteps: string;
    fromCache: string;
  };
}

// 时间范围选择器翻译（TimeRangePicker 复用）
// 观测模块跨信号翻译
export interface ObserveTranslations {
  freshness: {
    live: string;
    idle: string;
    stale: string;
    absent: string;
    unknown: string;
    idleHint: string;
    staleHint: string;
    justNow: string;
    minutesAgo: string;
    hoursAgo: string;
  };
}

export interface TimeRangePickerTranslations {
  last15min: string;
  last1h: string;
  last24h: string;
  last7d: string;
  last15d: string;
  last30d: string;
  timeRangePresets: string;
  timeRangeCustomRelative: string;
  timeRangeAbsolute: string;
  timeRangeLastN: string;
  timeRangeMinutes: string;
  timeRangeHours: string;
  timeRangeDays: string;
  timeRangeStart: string;
  timeRangeEnd: string;
  timeRangeApply: string;
  timeRangeInvalidRange: string;
}

// Log 翻译
export interface LogTranslations extends TimeRangePickerTranslations {
  latestLog: string;
  totalGlobal: string;
  topService: string;
  pageTitle: string;
  pageDescription: string;
  searchPlaceholder: string;
  // 分面
  facetServices: string;
  facetSeverities: string;
  facetScopes: string;
  // 列表 / 详情
  timestamp: string;
  service: string;
  severity: string;
  body: string;
  scopeName: string;
  traceId: string;
  spanId: string;
  attributes: string;
  resourceInfo: string;
  viewTrace: string;
  // 分页
  showMore: string;
  showing: string;
  // 状态
  noLogs: string;
  loading: string;
  loadFailed: string;
  noCluster: string;
  noClusterDesc: string;
  // 直方图 / 过滤器
  logVolume: string;
  clearFilters: string;
}

// APM 翻译
export interface ApmTranslations extends TimeRangePickerTranslations {
  /** 从 trace 跳到该链路的日志 */
  viewLogs: string;
  pageTitle: string;
  pageDescription: string;
  // services
  services: string;
  selectService: string;
  allServices: string;
  operations: string;
  selectOperation: string;
  allOperations: string;
  // trace list
  traces: string;
  traceId: string;
  rootService: string;
  rootOperation: string;
  duration: string;
  spans: string;
  serviceCount: string;
  startTime: string;
  noTraces: string;
  // trace detail
  traceDetail: string;
  waterfall: string;
  spanDetail: string;
  operationName: string;
  serviceName: string;
  status: string;
  parentSpan: string;
  childSpans: string;
  // filters
  minDuration: string;
  maxDuration: string;
  // time range (APM-specific, rest inherited from TimeRangePickerTranslations)
  last6h: string;
  // service list (Kibana-style)
  searchServices: string;
  namespace: string;
  tags: string;
  successRate: string;
  latencyAvg: string;
  throughput: string;
  errorRate: string;
  tpm: string;
  // service overview tabs
  overview: string;
  transactions: string;
  dependencies: string;
  errors: string;
  // tables
  impact: string;
  searchTransactions: string;
  viewAll: string;
  // service metrics
  totalRequests: string;
  // chart titles
  latencyChartTitle: string;
  throughputChartTitle: string;
  errorRateChartTitle: string;
  spanTypeChartTitle: string;
  // trace waterfall
  latencyDistribution: string;
  traceSample: string;
  currentSample: string;
  timeline: string;
  metadata: string;
  logs: string;
  // metadata tab
  serviceBreakdown: string;
  durationPercent: string;
  errorSpans: string;
  resourceOverview: string;
  instanceCount: string;
  traceOf: string;
  // span detail drawer
  spanKind: string;
  statusCode: string;
  selfTime: string;
  childTime: string;
  startOffset: string;
  resourceInfo: string;
  podName: string;
  nodeName: string;
  deploymentName: string;
  namespaceName: string;
  clusterName: string;
  serviceVersion: string;
  httpMethod: string;
  httpRoute: string;
  httpUrl: string;
  httpStatusCode: string;
  dbSystem: string;
  dbName: string;
  dbOperation: string;
  dbTable: string;
  dbStatement: string;
  spanIds: string;
  httpAttributes: string;
  dbAttributes: string;
  // topology
  serviceTopology: string;
  topoLatency: string;
  topoThroughput: string;
  topoErrorRate: string;
  topoCalls: string;
  nodeTypeService: string;
  nodeTypeIngress: string;
  nodeTypePod: string;
  nodeTypeNode: string;
  nodeTypeDatabase: string;
  nodeTypeExternal: string;
  legendHealthy: string;
  legendWarning: string;
  legendCritical: string;
  pauseAutoRefresh: string;
  resumeAutoRefresh: string;
  // states
  loading: string;
  loadFailed: string;
  noCluster: string;
  noClusterDesc: string;
  // trend charts
  latencyTrend: string;
  throughputTrend: string;
  errorCountTrend: string;
  // error traces
  errorTraces: string;
  noErrors: string;
  // http status distribution
  httpStatusDistribution: string;
  statusCode2xx: string;
  statusCode3xx: string;
  statusCode4xx: string;
  statusCode5xx: string;
  // database stats
  database: string;
  dbCallCount: string;
  noDBCalls: string;
  // slow traces
  slowTraces: string;
  noData: string;
  // error detail
  exceptionType: string;
  exceptionMessage: string;
  stacktrace: string;
  errorInfo: string;
  httpStatus: string;
  showStacktrace: string;
  // cross-signal correlation
  correlatedLogs: string;
  viewFullTrace: string;
  legendSelfTime: string;
  legendWaiting: string;
  focusLegendHint: string;
  errorSource: string;
  errorSourceSpanEvent: string;
  errorSourceStatusMessage: string;
  errorSourceTraceLog: string;
  logAttrExpand: string;
  latencyFilterActive: string;
  clearFilter: string;
  brushHint: string;
  correlations: string;
  correlationsDesc: string;
  correlationModeFailure: string;
  correlationModeLatency: string;
  corrAttrValue: string;
  corrFgRatioFailure: string;
  corrFgRatioLatency: string;
  corrBgRatio: string;
  corrLift: string;
  corrImpact: string;
  corrLowSample: string;
  corrNoSamples: string;
  corrThreshold: string;
  noCorrelatedLogs: string;
  viewCorrelatedLogs: string;
  // units
  us: string;
  ms: string;
  s: string;
}

// Event 页面翻译
export interface EventTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allSeverities: string;
  allKinds: string;
  allNamespaces: string;
  totalEvents: string;
  warningEvents: string;
  criticalEvents: string;
  normalEvents: string;
  reason: string;
  source: string;
  count: string;
  firstTime: string;
  lastTime: string;
  involvedObject: string;
  noEvents: string;
  noEventsDescription: string;
  loadFailed: string;
  // Drawer 详情
  overview: string;
  basicInfo: string;
  eventInfo: string;
  relatedResource: string;
}

// 部署管理页面翻译（/admin/deploy）
export interface DeployPageTranslations {
  pageTitle: string;
  pageDescription: string;
  // Config 仓库配置区
  configSection: string;
  configRepo: string;
  configRepoPlaceholder: string;
  editConfig: string;
  // 部署路径
  pathsSection: string;
  pathsSectionHint: string;
  pathCol: string;
  pathPlaceholder: string;
  addPath: string;
  deletePath: string;
  noPathsHint: string;
  // 调谐配置
  pollInterval: string;
  pollIntervalUnit: string;
  autoDeploy: string;
  autoDeployOn: string;
  autoDeployOff: string;
  autoDeployHint: string;
  testConnection: string;
  testConnectionSuccess: string;
  testConnectionFailed: string;
  // 当前状态区
  statusSection: string;
  pathStatusCol: string;
  resourceCount: string;
  statusCol: string;
  statusInSync: string;
  statusOutOfSync: string;
  statusPending: string;
  statusDeploying: string;
  statusFailed: string;
  statusNeverSynced: string;
  noStatus: string;
  noStatusHint: string;
  syncNow: string;
  // 部署历史区
  historySection: string;
  timeCol: string;
  pathCol2: string;
  triggerCol: string;
  commitCol: string;
  statusCol2: string;
  triggerAuto: string;
  triggerManual: string;
  triggerRollback: string;
  viewDetail: string;
  rollbackBtn: string;
  rollbackConfirm: string;
  rollbackSuccess: string;
  rollbackFailed: string;
  noHistory: string;
  noHistoryHint: string;
  // 部署详情弹窗
  detailTitle: string;
  detailPath: string;
  detailNamespace: string;
  detailTrigger: string;
  detailSourceRepo: string;
  detailCommit: string;
  detailCommitMessage: string;
  detailDeployedAt: string;
  detailDuration: string;
  detailResourceChanged: string;
  detailStatus: string;
  detailErrorMessage: string;
  detailAuthor: string;
  detailPR: string;
  detailChangedFiles: string;
  detailCompareLink: string;
  // 状态
  saveFailed: string;
  saveSuccess: string;
  loadFailed: string;
  // 前置条件
  githubNotConnected: string;
  githubNotConnectedHint: string;
  connectGithub: string;
}

// GitHub 集成页面翻译（/settings/github）
export interface GitHubPageTranslations {
  pageTitle: string;
  pageDescription: string;
  // Tabs
  tabConnection: string;
  tabDeploy: string;
  tabHistory: string;
  // 连接状態
  statusConnected: string;
  statusNotConnected: string;
  connectButton: string;
  disconnectButton: string;
  disconnectConfirm: string;
  connectedAs: string;
  // 已授权仓库
  reposSection: string;
  reposCount: string;
  noRepos: string;
  noReposHint: string;
  // 状態
  loadFailed: string;
  saveFailed: string;
}

// 数据源管理页面翻译
export interface DataSourceTranslations {
  pageTitle: string;
  pageDescription: string;
  mock: string;
  api: string;
  apiOnly: string;
  switchedToMock: string;
  switchedToApi: string;
  refreshHint: string;
  totalModules: string;
  mockCount: string;
  apiCount: string;
  groupObserve: string;
  groupCluster: string;
  groupAdmin: string;
  groupSettings: string;
  groupAiops: string;
}

// 完整翻译结构
export interface Translations {
  locale: "zh" | "ja";  // 语言标识符，用于日期格式化等
  nav: NavTranslations;
  common: CommonTranslations;
  status: StatusTranslations;
  audit: AuditTranslations;
  pod: PodTranslations;
  node: NodeTranslations;
  deployment: DeploymentTranslations;
  service: ServiceTranslations;
  namespace: NamespaceTranslations;
  ingress: IngressTranslations;
  alert: AlertTranslations;
  overview: OverviewTranslations;
  workbench: WorkbenchTranslations;
  users: UsersTranslations;
  roles: RolesTranslations;
  clusters: ClustersTranslations;
  agents: AgentsTranslations;
  notifications: NotificationsTranslations;
  login: LoginTranslations;
  confirm: ConfirmTranslations;
  table: TableTranslations;
  daemonset: DaemonSetTranslations;
  statefulset: StatefulSetTranslations;
  placeholder: PlaceholderTranslations;
  commands: CommandsTranslations;
  slo: SLOTranslations;
  aiSettingsPage: AISettingsPageTranslations;
  aiChatPage: AIChatPageTranslations;
  aboutPage: AboutTranslations;
  job: JobTranslations;
  cronjob: CronJobTranslations;
  storagePage: StorageTranslations;
  policyPage: PolicyTranslations;
  nodeMetrics: NodeMetricsTranslations;
  observe: ObserveTranslations;
  aiops: AIOpsTranslations;
  apm: ApmTranslations;
  logs: LogTranslations;
  dataSource: DataSourceTranslations;
  event: EventTranslations;
  deployPage: DeployPageTranslations;
  githubPage: GitHubPageTranslations;
}

// 国际化上下文
export interface I18nContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: Translations;
}
