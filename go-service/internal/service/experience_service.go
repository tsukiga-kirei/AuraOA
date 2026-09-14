package service

import (
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/repository"
)

// ExperienceService 组织审核互动、嵌入访问校验和管理员反馈查询。
type ExperienceService struct {
	repo  *repository.ExperienceRepo
	audit *AuditExecuteService
}

// NewExperienceService 创建用户体验服务。
func NewExperienceService(repo *repository.ExperienceRepo, audit *AuditExecuteService) *ExperienceService {
	return &ExperienceService{repo: repo, audit: audit}
}

func (s *ExperienceService) auditAccess(c *gin.Context, id uuid.UUID) (*model.AuditLog, error) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil || tenantID == uuid.Nil {
		return nil, newServiceError(errcode.ErrPermissionDenied, "缺少租户上下文")
	}
	log, err := s.audit.auditLogRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrPermissionDenied, "审核记录不存在或无权访问")
	}
	if c.GetBool("embed_mode") {
		if err = s.audit.checkEmbedAuditTaskAccess(c, tenantID, log); err != nil {
			return nil, err
		}
	}
	return log, nil
}

// viewer 只使用 OA 父页传入并经 OA 解析的人员身份，绝不使用嵌入后台执行账号署名。
// 返回: oaUserID, loginUsername, displayName, error
func (s *ExperienceService) viewer(c *gin.Context) (string, string, string, error) {
	id := strings.TrimSpace(c.GetHeader("X-Embed-OA-User-ID"))
	if id == "" {
		id = strings.TrimSpace(c.Query("oa_user_id"))
	}
	if id == "" {
		id = strings.TrimSpace(c.Query("oa_current_user_id"))
	}
	if id == "" || utf8.RuneCountInString(id) > 128 {
		return "", "", "", newServiceError(errcode.ErrPermissionDenied, "未识别到 OA 操作人，请从 OA 流程页面重新打开")
	}
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		return "", "", "", newServiceError(errcode.ErrPermissionDenied, "缺少租户上下文")
	}
	adapter, err := s.audit.getOAAdapter(c.Request.Context(), tenantID)
	if err != nil {
		return "", "", "", err
	}
	username, err := adapter.ResolveUsernameByOAUserID(c.Request.Context(), id)
	if err != nil || strings.TrimSpace(username) == "" {
		return "", "", "", newServiceError(errcode.ErrPermissionDenied, "无法识别 OA 操作人")
	}

	displayName, displayNameErr := adapter.ResolveUserDisplayNameByOAUserID(c.Request.Context(), id)
	if displayNameErr != nil {
		logger := pkglogger.Global()
		if tenant, tenantErr := s.audit.tenantRepo.FindByID(tenantID); tenantErr == nil {
			logger = pkglogger.GetTenantLogger(tenant.Code)
		}
		logger.Warn("审核体验评论：查询 OA 用户姓名失败，回退本地显示名或登录账号",
			zap.String("oaUserID", id),
			zap.String("username", username),
			zap.Error(displayNameErr))
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || displayName == username {
		var localUser model.User
		if err := s.audit.db.WithContext(c.Request.Context()).
			Table("users").
			Joins("JOIN org_members ON org_members.user_id = users.id").
			Where("org_members.tenant_id = ? AND users.username = ? AND users.status = 'active'", tenantID, username).
			First(&localUser).Error; err == nil && strings.TrimSpace(localUser.DisplayName) != "" {
			displayName = strings.TrimSpace(localUser.DisplayName)
		}
	}
	if displayName == "" {
		displayName = username
	}
	return id, username, displayName, nil
}

// Interactions 获取有权访问的审核互动；匿名嵌入访问可只读。
func (s *ExperienceService) Interactions(c *gin.Context, id uuid.UUID, q dto.ExperienceQuery) (*dto.AuditInteractionResponse, error) {
	if _, err := s.auditAccess(c, id); err != nil {
		return nil, err
	}
	oaID, _, _, identityErr := s.viewer(c)
	out, err := s.repo.Interactions(c, id, q)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取审核互动失败")
	}
	out.CanInteract = identityErr == nil
	for i := range out.Items {
		out.Items[i].CanManage = identityErr == nil && out.Items[i].OAUserID == oaID
	}
	return out, nil
}

// Comment 追加纯文本评论，不调用 AI。
func (s *ExperienceService) Comment(c *gin.Context, id uuid.UUID, req dto.AuditCommentRequest) (*model.AuditComment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" || utf8.RuneCountInString(content) > 2000 || (req.Feedback != nil && *req.Feedback != "like" && *req.Feedback != "dislike") {
		return nil, newServiceError(errcode.ErrParamValidation, "评论须为 1–2000 字，满意度只能为 like 或 dislike")
	}
	log, err := s.auditAccess(c, id)
	if err != nil {
		return nil, err
	}
	if log.Status != model.JobStatusCompleted {
		return nil, newServiceError(errcode.ErrParamValidation, "审核完成后才能评论")
	}
	oaID, _, displayName, err := s.viewer(c)
	if err != nil {
		return nil, err
	}
	row := &model.AuditComment{ID: uuid.New(), TenantID: log.TenantID, AuditLogID: id, OAUserID: oaID, Username: displayName, Content: content, Feedback: req.Feedback, CreatedAt: apptime.Now(), UpdatedAt: apptime.Now(), CanManage: true}
	if err = s.repo.CreateComment(c, row); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "保存评论失败")
	}
	return row, nil
}

// List 返回租户内有互动的审核或被评价的助手消息。
func (s *ExperienceService) List(c *gin.Context, kind string, q dto.ExperienceQuery) (interface{}, error) {
	if id, err := uuid.Parse(c.GetString("tenant_id")); err != nil || id == uuid.Nil {
		return nil, newServiceError(errcode.ErrPermissionDenied, "缺少租户上下文")
	}
	var data interface{}
	var err error
	if kind == "audit" {
		data, err = s.repo.ListAudits(c, q)
	} else {
		data, err = s.repo.ListAgents(c, q)
	}
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取体验反馈失败")
	}
	return data, nil
}

// AuditDetail 返回管理员查看的原审核结论与分页评论。
func (s *ExperienceService) AuditDetail(c *gin.Context, id uuid.UUID, q dto.ExperienceQuery) (interface{}, error) {
	log, err := s.auditAccess(c, id)
	if err != nil {
		return nil, err
	}
	interactions, err := s.repo.Interactions(c, id, q)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取审核互动失败")
	}
	return gin.H{"audit_result": buildAuditResultFromLog(log), "interactions": interactions}, nil
}

// ManageComment 编辑或删除本人评论；写入条件同时校验租户、审核批次与作者。
func (s *ExperienceService) ManageComment(c *gin.Context, auditID, commentID uuid.UUID, req *dto.AuditCommentRequest) error {
	if req != nil {
		req.Content = strings.TrimSpace(req.Content)
		if req.Content == "" || utf8.RuneCountInString(req.Content) > 2000 || (req.Feedback != nil && *req.Feedback != "like" && *req.Feedback != "dislike") {
			return newServiceError(errcode.ErrParamValidation, "评论内容或满意度无效")
		}
	}
	if _, err := s.auditAccess(c, auditID); err != nil {
		return err
	}
	oaID, _, _, err := s.viewer(c)
	if err != nil {
		return err
	}
	var changed bool
	if req == nil {
		changed, err = s.repo.DeleteOwnComment(c, auditID, commentID, oaID)
	} else {
		changed, err = s.repo.UpdateOwnComment(c, auditID, commentID, oaID, *req)
	}
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "更新评论失败")
	}
	if !changed {
		return newServiceError(errcode.ErrPermissionDenied, "评论不存在或无权操作")
	}
	return nil
}
