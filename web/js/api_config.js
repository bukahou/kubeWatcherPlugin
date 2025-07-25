// js/api_config.js
let API_BASE_URL = "";

const currentHost = window.location.hostname;
const currentProtocol = window.location.protocol;

if (currentHost === "127.0.0.1") {
  // 本地测试：显式设定本地 API 地址（端口根据后端服务实际情况修改）
  API_BASE_URL = "http://localhost:8081";
} else if (/^192\.168\.\d{1,3}\.\d{1,3}$/.test(currentHost)) {
  // 局域网部署：IP + 端口
  API_BASE_URL = `${currentProtocol}//${currentHost}:30080`;
} else {
  // 公网部署：默认走当前域名反向代理
  API_BASE_URL = `${currentProtocol}//${currentHost}`;
}

const API_ENDPOINTS = {
  cluster: {
    overview: `${API_BASE_URL}/uiapi/cluster/overview`,
  },
  deployment: {
    listAll: `${API_BASE_URL}/uiapi/deployment/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/deployment/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/deployment/get/${ns}/${name}`,
    listUnavailable: `${API_BASE_URL}/uiapi/deployment/list/unavailable`,
    listProgressing: `${API_BASE_URL}/uiapi/deployment/list/progressing`,
    scale: `${API_BASE_URL}/uiapi/deployment-ops/scale`,
  },
  event: {
    listAll: `${API_BASE_URL}/uiapi/event/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/event/list/by-namespace/${ns}`,
    listByObject: (ns, kind, name) =>
      `${API_BASE_URL}/uiapi/event/list/by-object/${ns}/${kind}/${name}`,
    summaryByType: `${API_BASE_URL}/uiapi/event/summary/type`,
    listRecent: (days) =>
      `${API_BASE_URL}/uiapi/event/list/recent?days=${days}`,
  },
  ingress: {
    listAll: `${API_BASE_URL}/uiapi/ingress/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/ingress/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/ingress/get/${ns}/${name}`,
    listReady: `${API_BASE_URL}/uiapi/ingress/list/ready`,
  },
  namespace: {
    list: `${API_BASE_URL}/uiapi/namespace/list`,
    get: (name) => `${API_BASE_URL}/uiapi/namespace/get/${name}`,
    listActive: `${API_BASE_URL}/uiapi/namespace/list/active`,
    listTerminating: `${API_BASE_URL}/uiapi/namespace/list/terminating`,
    summaryStatus: `${API_BASE_URL}/uiapi/namespace/summary/status`,
  },
  node: {
    list: `${API_BASE_URL}/uiapi/node/list`,
    metrics: `${API_BASE_URL}/uiapi/node/metrics`,
    overview: `${API_BASE_URL}/uiapi/node/overview`,
    getByName: (name) => `${API_BASE_URL}/uiapi/node/get/${name}`,
    schedule: `${API_BASE_URL}/uiapi/node-ops/schedule`,
  },
  pod: {
    listAll: `${API_BASE_URL}/uiapi/pod/list`,
    listByNamespace: (ns) => `${API_BASE_URL}/uiapi/pod/list/${ns}`,
    summary: `${API_BASE_URL}/uiapi/pod/summary`,
    usage: `${API_BASE_URL}/uiapi/pod/usage`,
    // ✅ 新增：获取简略 Pod 列表（PodInfo 结构）
    listBrief: `${API_BASE_URL}/uiapi/pod/list/brief`,
    describe: (ns, name) => `${API_BASE_URL}/uiapi/pod/describe/${ns}/${name}`,
    restart: (namespace, name) =>
      `${API_BASE_URL}/uiapi/pod-ops/restart/${namespace}/${name}`, // ✅ 新增重启接口
    logs: (namespace, name) =>
      `${API_BASE_URL}/uiapi/pod/logs/${namespace}/${name}`,
  },
  configmap: {
    listAll: `${API_BASE_URL}/uiapi/configmap/list`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/configmap/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/configmap/get/${ns}/${name}`,
    // ✅ 告警系统配置
    getAlertSettings: `${API_BASE_URL}/uiapi/configmap/alert/get`, // 获取配置
    updateSlack: `${API_BASE_URL}/uiapi/configmap/alert/slack`, // 更新 Slack
    updateWebhook: `${API_BASE_URL}/uiapi/configmap/alert/webhook`, // 更新 Webhook 开关
    updateMail: `${API_BASE_URL}/uiapi/configmap/alert/mail`, // 更新 Mail（含多人）
  },
  service: {
    listAll: `${API_BASE_URL}/uiapi/service/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/service/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/service/get/${ns}/${name}`,
    listExternal: `${API_BASE_URL}/uiapi/service/list/external`,
    listHeadless: `${API_BASE_URL}/uiapi/service/list/headless`,
  },
  auth: {
    login: `${API_BASE_URL}/uiapi/auth/login`, // 登录接口
    register: `${API_BASE_URL}/uiapi/auth/user/register`, // 注册用户
    updateRole: `${API_BASE_URL}/uiapi/auth/user/update-role`, // 修改用户角色
    listUsers: `${API_BASE_URL}/uiapi/auth/user/list`, // 获取用户列表
  },
};
