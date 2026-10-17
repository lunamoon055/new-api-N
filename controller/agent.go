package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type agentApplicationRequest struct {
	Note string `json:"note"`
}

type agentReviewRequest struct {
	Decision string `json:"decision" binding:"required"`
	Note     string `json:"note"`
}

type referralRewardRequest struct {
	TaskRecordID   int64  `json:"task_record_id" binding:"required"`
	SourceOrderID  string `json:"source_order_id" binding:"required"`
	CostEvidenceID string `json:"cost_evidence_id" binding:"required"`
	SaleUnits      *int64 `json:"sale_units" binding:"required"`
	CostUnits      *int64 `json:"cost_units" binding:"required"`
	ExtraCostUnits *int64 `json:"extra_cost_units" binding:"required"`
}

type referralPayoutRequest struct {
	PayoutReference string `json:"payout_reference" binding:"required"`
}

func GetAgentCenter(c *gin.Context) {
	userID := c.GetInt("id")
	application, err := model.GetAgentApplication(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to load application"})
		return
	}
	data := gin.H{"application": application, "agent_code": "", "promotion_code": "", "customer_count": 0,
		"reward_pending_units": int64(0), "reward_paid_units": int64(0)}
	var currentUser model.User
	if err := model.DB.Select("aff_code").First(&currentUser, userID).Error; err == nil {
		data["promotion_code"] = currentUser.AffCode
	}
	var totals []struct {
		Status string
		Amount int64
	}
	if err := model.DB.Model(&model.ReferralReward{}).
		Select("status, SUM(reward_units) AS amount").Where("referrer_id = ?", userID).
		Group("status").Scan(&totals).Error; err == nil {
		for _, total := range totals {
			if total.Status == model.ReferralRewardPending {
				data["reward_pending_units"] = total.Amount
			} else if total.Status == model.ReferralRewardPaid {
				data["reward_paid_units"] = total.Amount
			}
		}
	}
	if application != nil && application.Status == model.AgentStatusApproved {
		data["agent_code"] = currentUser.AffCode
		var count int64
		if err := model.DB.Model(&model.User{}).Where("inviter_id = ? AND referral_kind = ?", userID, "agent").Count(&count).Error; err == nil {
			data["customer_count"] = count
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func ApplyForAgent(c *gin.Context) {
	var request agentApplicationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid application"})
		return
	}
	request.Note = strings.TrimSpace(request.Note)
	if len([]rune(request.Note)) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Application note is too long"})
		return
	}
	if err := model.SubmitAgentApplication(c.GetInt("id"), request.Note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminListAgentApplications(c *gin.Context) {
	status := c.Query("status")
	if status != "" && status != model.AgentStatusPending && status != model.AgentStatusApproved && status != model.AgentStatusRejected && status != model.AgentStatusRevoked {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid status"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	items, total, err := model.ListAgentApplications(status, 50, (page-1)*50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to load applications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": total}})
}

func AdminReviewAgentApplication(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid application ID"})
		return
	}
	var request agentReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid decision"})
		return
	}
	request.Note = strings.TrimSpace(request.Note)
	if len([]rune(request.Note)) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Review note is too long"})
		return
	}
	if err := model.ReviewAgentApplication(id, c.GetInt("id"), request.Decision, request.Note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ListMyReferralRewards(c *gin.Context) {
	items, total, err := model.ListReferralRewards(c.GetInt("id"), "", 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to load rewards"})
		return
	}
	publicItems := make([]gin.H, 0, len(items))
	for _, item := range items {
		publicItems = append(publicItems, gin.H{
			"id": item.ID, "model_name": item.ModelName,
			"reward_units": item.RewardUnits, "status": item.Status,
			"eligible_at": item.EligibleAt, "created_at": item.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": publicItems, "total": total}})
}

func AdminListReferralRewards(c *gin.Context) {
	status := c.Query("status")
	if status != "" && status != model.ReferralRewardPending && status != model.ReferralRewardPaid && status != model.ReferralRewardReversed {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid status"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	items, total, err := model.ListReferralRewards(0, status, 50, (page-1)*50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to load rewards"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": total}})
}

func AdminCreateReferralReward(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var request referralRewardRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid reward data"})
		return
	}
	reward, err := model.CreateReferralRewardForTask(c.GetInt("id"), request.TaskRecordID, request.SourceOrderID, request.CostEvidenceID, *request.SaleUnits, *request.CostUnits, *request.ExtraCostUnits)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reward})
}

func AdminLookupReferralTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid task ID"})
		return
	}
	var task model.Task
	if err := model.DB.Select("id", "task_id", "user_id", "channel_id", "quota", "status", "billing_status", "finish_time", "properties").First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Task not found"})
		return
	}
	var user model.User
	_ = model.DB.Select("id", "inviter_id", "referral_kind").First(&user, task.UserId).Error
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id": task.ID, "task_id": task.TaskID, "user_id": task.UserId,
		"channel_id": task.ChannelId, "model_name": task.Properties.ModelName(),
		"quota": task.Quota, "status": task.Status, "billing_status": task.BillingStatus,
		"finish_time": task.FinishTime, "inviter_id": user.InviterId,
		"referral_kind": user.ReferralKind,
	}})
}

func AdminSearchReferralTasks(c *gin.Context) {
	taskID := strings.TrimSpace(c.Query("task_id"))
	if taskID == "" || len(taskID) > 191 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid task ID"})
		return
	}
	var tasks []model.Task
	if err := model.DB.Select("id", "task_id", "user_id", "channel_id", "quota", "status", "billing_status", "finish_time", "properties").
		Where("task_id = ?", taskID).Order("id desc").Limit(10).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Task lookup failed"})
		return
	}
	items := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, gin.H{
			"id": task.ID, "task_id": task.TaskID, "user_id": task.UserId,
			"channel_id": task.ChannelId, "model_name": task.Properties.ModelName(),
			"quota": task.Quota, "status": task.Status, "billing_status": task.BillingStatus,
			"finish_time": task.FinishTime,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func AdminMarkReferralRewardPaid(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid reward ID"})
		return
	}
	var request referralPayoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Payout reference required"})
		return
	}
	if err := model.MarkReferralRewardPaid(id, request.PayoutReference); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
