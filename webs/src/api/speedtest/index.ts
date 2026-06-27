import request from "@/utils/request";

export function listHomeAgents() {
  return request({
    url: "/api/v1/speedtest/agents",
    method: "get",
  });
}

export function createHomeAgent(data: any) {
  return request({
    url: "/api/v1/speedtest/agents/create",
    method: "post",
    data,
    headers: { "Content-Type": "multipart/form-data" },
  });
}

export function setHomeAgentMode(data: any) {
  return request({
    url: "/api/v1/speedtest/agents/mode",
    method: "post",
    data,
    headers: { "Content-Type": "multipart/form-data" },
  });
}

export function deleteHomeAgent(id: number) {
  return request({
    url: "/api/v1/speedtest/agents",
    method: "delete",
    params: { id },
  });
}

export function createSpeedTask(data: any) {
  return request({
    url: "/api/v1/speedtest/tasks",
    method: "post",
    data,
    headers: { "Content-Type": "multipart/form-data" },
  });
}

export function listSpeedTasks(params?: any) {
  return request({
    url: "/api/v1/speedtest/tasks",
    method: "get",
    params,
  });
}

export function cancelSpeedTasks(data?: any) {
  return request({
    url: "/api/v1/speedtest/tasks/cancel",
    method: "post",
    data,
    headers: { "Content-Type": "multipart/form-data" },
  });
}
