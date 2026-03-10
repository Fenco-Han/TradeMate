import { createRouter, createWebHistory } from "vue-router";
import AuditLogsPage from "./views/AuditLogsPage.vue";
import ApprovalsPage from "./views/ApprovalsPage.vue";
import DashboardPage from "./views/DashboardPage.vue";
import GoalsPage from "./views/GoalsPage.vue";
import LoginPage from "./views/LoginPage.vue";
import NotificationsPage from "./views/NotificationsPage.vue";
import TasksPage from "./views/TasksPage.vue";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      redirect: "/dashboard"
    },
    {
      path: "/login",
      component: LoginPage
    },
    {
      path: "/dashboard",
      component: DashboardPage
    },
    {
      path: "/goals",
      component: GoalsPage
    },
    {
      path: "/approvals",
      component: ApprovalsPage
    },
    {
      path: "/tasks",
      component: TasksPage
    },
    {
      path: "/notifications",
      component: NotificationsPage
    },
    {
      path: "/audit-logs",
      component: AuditLogsPage
    }
  ]
});
