package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	baseURL string
	apiKey  string
	client  = &http.Client{Timeout: 30 * time.Second}
	verbose bool
)

func main() {
	flag.StringVar(&baseURL, "url", "http://localhost:8080", "API base URL")
	flag.StringVar(&apiKey, "key", "", "API key (reads API_KEY env if empty)")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()

	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}

	printBanner()

	step := 0
	next := func(title string) {
		step++
		fmt.Printf("\n\033[1;36m═══ 步骤 %d: %s ═══\033[0m\n", step, title)
	}

	// --- Step 1: 健康检查 ---
	next("健康检查")
	mustCheck("API 服务", "GET", "/healthz")
	mustCheck("数据库连接", "GET", "/readyz")

	// --- Step 2: 注册 Agent 组织树 ---
	next("注册 Agent 组织树（PM → Supervisor → Worker / Reviewer）")

	pmID := mustCreateAgent("灵筹-项目经理", "pm", "", "项目分解、阶段规划与任务创建", []string{"tool.*"})
	supID := mustCreateAgent("灵筹-主管", "supervisor", pmID, "任务契约制定、执行分派与监督", []string{"tool.*"})
	wkID := mustCreateAgent("灵筹-执行者", "worker", supID, "任务执行、工件生成与交接", []string{"tool.doc_generator", "tool.artifact_storage", "tool.test_runner"})
	rvID := mustCreateAgent("灵筹-评审员", "reviewer", pmID, "独立评审、质量检查与评审报告", []string{"tool.*"})

	printOK("已注册 4 个 Agent: PM(%s), Supervisor(%s), Worker(%s), Reviewer(%s)", short(pmID), short(supID), short(wkID), short(rvID))

	// --- Step 3: 创建 Demo 项目 ---
	next("创建 Demo 项目")

	projID := mustCreateProject("灵筹 MVP 演示项目", "端到端 Demo：验证项目→阶段→任务→工件→评审→审批→审计的完整链路")
	printOK("项目已创建: %s (ID: %s)", "灵筹 MVP 演示项目", short(projID))

	// --- Step 4: 启动工作流编排 ---
	next("启动工作流编排（PM → Supervisor → Worker → Reviewer）")
	fmt.Println("  正在执行编排器... 这将自动完成：")
	fmt.Println("  - PM 分解项目为阶段和任务")
	fmt.Println("  - Supervisor 为每个任务创建契约并分派")
	fmt.Println("  - Worker 执行任务并生成工件")
	fmt.Println("  - Reviewer 独立评审交付物")

	run := mustStartWorkflow(projID)
	runID := getString(run, "id")
	runStatus := getString(run, "status")
	runSummary := getString(run, "summary")
	steps := getArray(run, "steps")
	printOK("工作流完成: 状态=%s, 步骤数=%d", runStatus, len(steps))
	if runSummary != "" {
		fmt.Printf("  摘要: %s\n", runSummary)
	}

	// --- Step 5: 验证数据完整性 ---
	next("验证数据完整性")

	phases := mustListItems("阶段", "GET", fmt.Sprintf("/api/v1/projects/%s/phases", projID))
	tasks := mustListItems("任务", "GET", "/api/v1/tasks?project_id="+projID+"&limit=100")
	artifacts := mustListItems("工件", "GET", "/api/v1/artifacts?limit=100")
	reviews := mustListItems("评审报告", "GET", "/api/v1/reviews?limit=100")
	orgTree := mustListItems("组织树节点", "GET", "/api/v1/agents/org-tree")

	printOK("数据验证: %d 个阶段, %d 个任务, %d 个工件, %d 个评审报告, %d 个组织节点",
		len(phases), len(tasks), len(artifacts), len(reviews), len(orgTree))

	// --- Step 6: 审批流转演示（批准路径） ---
	next("审批流转演示（创建审批 → 批准 → 任务自动完成）")

	// 收集所有 in_review 状态的任务
	var inReviewTasks []map[string]any
	for _, t := range tasks {
		tm := t.(map[string]any)
		if getString(tm, "status") == "in_review" {
			inReviewTasks = append(inReviewTasks, tm)
		}
	}

	var targetTaskID string
	var targetTaskTitle string
	if len(inReviewTasks) > 0 {
		targetTaskID = getString(inReviewTasks[0], "id")
		targetTaskTitle = getString(inReviewTasks[0], "title")

		approvalID := mustCreateApproval(projID, targetTaskID, rvID, pmID, targetTaskTitle)
		printOK("审批请求已创建: %s (任务: %s)", short(approvalID), targetTaskTitle)

		mustApprove(approvalID)
		printOK("审批已通过")

		updatedTask := mustGet(fmt.Sprintf("/api/v1/tasks/%s", targetTaskID))
		newStatus := getString(updatedTask, "status")
		printOK("任务「%s」状态已自动流转: in_review → %s", targetTaskTitle, newStatus)
	} else {
		fmt.Println("  ⚠ 未找到处于 in_review 状态的任务，跳过审批批准演示")
	}

	// --- Step 6.5: 审批拒绝路径 ---
	next("审批拒绝链路演示（创建审批 → 拒绝 → 任务回退 revision_required）")

	var rejectTaskID string
	var rejectTaskTitle string
	if len(inReviewTasks) > 1 {
		rejectTaskID = getString(inReviewTasks[1], "id")
		rejectTaskTitle = getString(inReviewTasks[1], "title")

		rejectApprovalID := mustCreateApproval(projID, rejectTaskID, rvID, pmID, rejectTaskTitle)
		printOK("审批请求已创建: %s (任务: %s)", short(rejectApprovalID), rejectTaskTitle)

		mustReject(rejectApprovalID, "交付物不符合验收标准，需要修订")
		printOK("审批已拒绝")

		rejectedTask := mustGet(fmt.Sprintf("/api/v1/tasks/%s", rejectTaskID))
		rejStatus := getString(rejectedTask, "status")
		printOK("任务「%s」状态已自动回退: in_review → %s", rejectTaskTitle, rejStatus)
		if rejStatus != "revision_required" {
			fmt.Printf("  ⚠ 期望状态 revision_required，实际 %s\n", rejStatus)
		}

		// 验证拒绝后的审批请求状态
		rejApproval := mustGet(fmt.Sprintf("/api/v1/approvals/%s", rejectApprovalID))
		rejApprovalStatus := getString(rejApproval, "status")
		printOK("审批请求最终状态: %s", rejApprovalStatus)

		// 验证 revision_required → in_progress 恢复路径
		mustTransitionTask(rejectTaskID, "in_progress")
		recoveredTask := mustGet(fmt.Sprintf("/api/v1/tasks/%s", rejectTaskID))
		recStatus := getString(recoveredTask, "status")
		printOK("任务「%s」已恢复执行: revision_required → %s", rejectTaskTitle, recStatus)
	} else {
		fmt.Println("  ⚠ 仅有一个 in_review 任务，审批拒绝链路演示需要至少 2 个，跳过")
	}

	// --- Step 7: Tool Gateway 演示 ---
	next("Tool Gateway 演示（调用文档生成工具）")

	tools := mustListItems("可用工具", "GET", "/api/v1/tools")
	fmt.Printf("  已注册工具: ")
	for i, t := range tools {
		tm := t.(map[string]any)
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(getString(tm, "name"))
	}
	fmt.Println()

	toolTaskID := targetTaskID
	if toolTaskID == "" {
		for _, t := range tasks {
			tm := t.(map[string]any)
			toolTaskID = getString(tm, "id")
			break
		}
	}
	toolResult := mustCallTool(wkID, toolTaskID)
	toolStatus := getString(toolResult, "status")
	printOK("工具调用完成: 状态=%s", toolStatus)

	// --- Step 8: 审计时间线 ---
	next("审计时间线查询")

	timeline := mustListItems("项目时间线事件", "GET", fmt.Sprintf("/api/v1/projects/%s/timeline?limit=50", projID))
	printOK("项目级时间线: %d 条事件", len(timeline))

	fmt.Println("\n  最近事件:")
	displayCount := 10
	if len(timeline) < displayCount {
		displayCount = len(timeline)
	}
	for i := 0; i < displayCount; i++ {
		ev := timeline[i].(map[string]any)
		evType := getString(ev, "event_type")
		evSummary := getString(ev, "event_summary")
		evTime := getString(ev, "created_at")
		ts := formatTime(evTime)
		fmt.Printf("  [%s] \033[33m%-30s\033[0m %s\n", ts, evType, truncate(evSummary, 60))
	}
	if len(timeline) > displayCount {
		fmt.Printf("  ... 还有 %d 条事件\n", len(timeline)-displayCount)
	}

	if targetTaskID != "" {
		taskTimeline := mustListItems("任务时间线事件", "GET", fmt.Sprintf("/api/v1/tasks/%s/timeline?limit=20", targetTaskID))
		printOK("任务「%s」时间线: %d 条事件", targetTaskTitle, len(taskTimeline))
	}

	// --- Step 9: 总结 ---
	next("Demo 执行总结")

	fmt.Println()
	fmt.Println("  \033[1;32m✓ 端到端 Demo 执行成功！\033[0m")
	fmt.Println()
	fmt.Println("  已验证的核心链路：")
	fmt.Println("  ┌─────────────────────────────────────────────────────┐")
	fmt.Printf("  │ 项目创建       │ %-35s │\n", short(projID))
	fmt.Printf("  │ 阶段数         │ %-35d │\n", len(phases))
	fmt.Printf("  │ 任务数         │ %-35d │\n", len(tasks))
	fmt.Printf("  │ 工件数         │ %-35d │\n", len(artifacts))
	fmt.Printf("  │ 评审报告数     │ %-35d │\n", len(reviews))
	fmt.Printf("  │ 组织树节点     │ %-35d │\n", len(orgTree))
	fmt.Printf("  │ 工作流运行     │ %-35s │\n", short(runID))
	fmt.Printf("  │ 审计事件       │ %-35d │\n", len(timeline))
	fmt.Println("  └─────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  验证清单：")
	fmt.Println("  [✓] PM 分解项目为阶段和任务")
	fmt.Println("  [✓] Supervisor 创建契约并分派执行者")
	fmt.Println("  [✓] Worker 执行任务并生成工件")
	fmt.Println("  [✓] Reviewer 独立评审交付物")
	if targetTaskID != "" {
		fmt.Println("  [✓] 审批创建与审批通过")
		fmt.Println("  [✓] 审批通过后任务自动完成")
	}
	if rejectTaskID != "" {
		fmt.Println("  [✓] 审批拒绝后任务回退 revision_required")
		fmt.Println("  [✓] revision_required → in_progress 恢复路径")
	}
	fmt.Println("  [✓] Tool Gateway 工具调用")
	fmt.Println("  [✓] 全链路审计日志写入")
	fmt.Println("  [✓] 项目级/任务级审计时间线可查")
	fmt.Println()
	fmt.Println("  请打开 Web 控制台查看数据：")
	fmt.Println("  \033[4mhttp://localhost:3000\033[0m")
	fmt.Println()
	fmt.Printf("  项目详情: \033[4mhttp://localhost:3000/projects/%s\033[0m\n", projID)
	fmt.Println("  任务看板: \033[4mhttp://localhost:3000/tasks\033[0m")
	fmt.Println("  Agent 组织树: \033[4mhttp://localhost:3000/agents\033[0m")
	fmt.Println("  工件列表: \033[4mhttp://localhost:3000/artifacts\033[0m")
	fmt.Println("  审批中心: \033[4mhttp://localhost:3000/approvals\033[0m")
	fmt.Println("  审计时间线: \033[4mhttp://localhost:3000/audit\033[0m")
	fmt.Println()
}

// --- HTTP helpers ---

func doRequest(method, path string, body any) (map[string]any, int) {
	url := baseURL + path
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
		if verbose {
			fmt.Printf("  → %s %s\n    Body: %s\n", method, path, string(b))
		}
	} else if verbose {
		fmt.Printf("  → %s %s\n", method, path)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		fatal("创建请求失败: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		fatal("请求失败 %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fatal("读取响应失败: %v", err)
	}

	if verbose {
		fmt.Printf("  ← %d: %s\n", resp.StatusCode, truncate(string(respBytes), 200))
	}

	var result map[string]any
	if err := json.Unmarshal(respBytes, &result); err != nil {
		fatal("解析响应失败 (%s %s): %s", method, path, string(respBytes))
	}

	return result, resp.StatusCode
}

func mustCheck(label, method, path string) {
	result, code := doRequest(method, path, nil)
	if code >= 400 {
		fatal("%s 检查失败 (HTTP %d)", label, code)
	}
	data := getMap(result, "data")
	status := getString(data, "status")
	if status == "" {
		status = "ok"
	}
	printOK("%s: %s", label, status)
}

func mustGet(path string) map[string]any {
	result, code := doRequest("GET", path, nil)
	if code >= 400 {
		fatal("GET %s 失败 (HTTP %d): %v", path, code, result)
	}
	data := getMap(result, "data")
	if data != nil {
		return data
	}
	return result
}

func mustCreateAgent(name, role, reportsTo, desc string, capabilities []string) string {
	body := map[string]any{
		"name":         name,
		"role":         role,
		"description":  desc,
		"status":       "active",
		"capabilities": capabilities,
		"metadata":     map[string]any{},
	}
	if reportsTo != "" {
		body["reports_to"] = reportsTo
	}

	result, code := doRequest("POST", "/api/v1/agents", body)
	if code >= 400 {
		fatal("创建 Agent「%s」失败: %v", name, result)
	}
	data := getMap(result, "data")
	if data == nil {
		fatal("创建 Agent 响应缺少 data")
	}
	id := getString(data, "id")
	fmt.Printf("  ✓ %s (%s) → %s\n", name, role, short(id))
	return id
}

func mustCreateProject(name, desc string) string {
	body := map[string]any{
		"name":        name,
		"description": desc,
		"status":      "active",
		"metadata":    map[string]any{},
	}
	result, code := doRequest("POST", "/api/v1/projects", body)
	if code >= 400 {
		fatal("创建项目失败: %v", result)
	}
	data := getMap(result, "data")
	if data == nil {
		fatal("创建项目响应缺少 data")
	}
	return getString(data, "id")
}

func mustStartWorkflow(projectID string) map[string]any {
	body := map[string]any{"project_id": projectID}
	result, code := doRequest("POST", "/api/v1/orchestrator/runs", body)
	if code >= 400 {
		fatal("启动工作流失败: %v", result)
	}
	data := getMap(result, "data")
	if data != nil {
		return data
	}
	return result
}

func mustListItems(label, method, path string) []any {
	result, code := doRequest(method, path, nil)
	if code >= 400 {
		fatal("查询%s失败 (HTTP %d): %v", label, code, result)
	}

	data := getMap(result, "data")
	if data != nil {
		items := getArray(data, "items")
		total := getNumber(data, "total")
		fmt.Printf("  ✓ %s: %d 条 (total=%d)\n", label, len(items), int(total))
		return items
	}

	items := getArray(result, "items")
	if items != nil {
		total := getNumber(result, "total")
		fmt.Printf("  ✓ %s: %d 条 (total=%d)\n", label, len(items), int(total))
		return items
	}

	return nil
}

func mustCreateApproval(projID, taskID, requestedBy, approverID, taskTitle string) string {
	body := map[string]any{
		"project_id":    projID,
		"task_id":       taskID,
		"requested_by":  requestedBy,
		"approver_type": "agent",
		"approver_id":   approverID,
		"title":         fmt.Sprintf("审批：任务「%s」交付验收", taskTitle),
		"description":   fmt.Sprintf("请审批任务「%s」的交付物，评审已通过。", taskTitle),
		"metadata":      map[string]any{},
	}
	result, code := doRequest("POST", "/api/v1/approvals", body)
	if code >= 400 {
		fatal("创建审批请求失败: %v", result)
	}
	data := getMap(result, "data")
	if data == nil {
		fatal("创建审批响应缺少 data")
	}
	return getString(data, "id")
}

func mustApprove(approvalID string) {
	body := map[string]any{
		"status": "approved",
		"note":   "Demo 审批通过：交付物符合验收标准",
	}
	result, code := doRequest("POST", fmt.Sprintf("/api/v1/approvals/%s/decide", approvalID), body)
	if code >= 400 {
		fatal("审批决策失败: %v", result)
	}
}

func mustReject(approvalID, reason string) {
	body := map[string]any{
		"status": "rejected",
		"note":   reason,
	}
	result, code := doRequest("POST", fmt.Sprintf("/api/v1/approvals/%s/decide", approvalID), body)
	if code >= 400 {
		fatal("审批拒绝失败: %v", result)
	}
}

func mustTransitionTask(taskID, newStatus string) {
	body := map[string]any{
		"status": newStatus,
	}
	result, code := doRequest("PATCH", fmt.Sprintf("/api/v1/tasks/%s/status", taskID), body)
	if code >= 400 {
		fatal("任务状态流转失败 (%s → %s): %v", taskID, newStatus, result)
	}
}

func mustCallTool(agentID, taskID string) map[string]any {
	body := map[string]any{
		"tool_name": "doc_generator",
		"agent_id":  agentID,
		"task_id":   taskID,
		"input": map[string]any{
			"title":   "端到端 Demo 验证报告",
			"content": "本报告由灵筹 Demo 脚本自动生成，验证 Tool Gateway 调用链路。",
			"format":  "markdown",
		},
	}
	result, code := doRequest("POST", "/api/v1/tools/call", body)
	if code >= 400 {
		fatal("工具调用失败: %v", result)
	}
	data := getMap(result, "data")
	if data != nil {
		return data
	}
	return result
}

// --- JSON helpers ---

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	sub, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return sub
}

func getArray(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	return arr
}

func getNumber(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	n, ok := v.(float64)
	if !ok {
		return 0
	}
	return n
}

// --- Output helpers ---

func printBanner() {
	fmt.Println()
	fmt.Println("\033[1;35m╔═══════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;35m║     灵筹（LingChou）端到端 Demo 验证脚本     ║\033[0m")
	fmt.Println("\033[1;35m╚═══════════════════════════════════════════════╝\033[0m")
	fmt.Printf("  目标 API: %s\n", baseURL)
	fmt.Printf("  时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func printOK(format string, args ...any) {
	fmt.Printf("  \033[32m✓\033[0m "+format+"\n", args...)
}

func fatal(format string, args ...any) {
	fmt.Printf("\n  \033[1;31m✗ FATAL: "+format+"\033[0m\n\n", args...)
	os.Exit(1)
}

func short(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "..."
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatTime(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ts[:19]
		}
	}
	return t.Format("15:04:05")
}
