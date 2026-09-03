package handler

import (
	"net/http"

	"exam-server/internal/middleware"
	"exam-server/internal/model"
	"exam-server/internal/service"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	importService *service.ImportService
}

func NewImportHandler(importService *service.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

// PreviewExcel 接收上传的 Excel 文件，返回解析预览与校验数据
func (h *ImportHandler) PreviewExcel(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "请选择要上传的 Excel/CSV 文件"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "无法打开上传文件: "+err.Error()))
		return
	}
	defer file.Close()

	result, err := h.importService.ParseExcelPreview(file, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

// ConfirmImport 用户在前端核对预览并确认题库名称后正式写入数据库
func (h *ImportHandler) ConfirmImport(c *gin.Context) {
	var userID uint
	if uid, err := middleware.GetCurrentUserID(c); err == nil {
		userID = uid
	}

	var req service.ConfirmImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "参数格式错误: "+err.Error()))
		return
	}

	bank, err := h.importService.ConfirmImport(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(bank))
}

// DownloadTemplate 提供标准导入格式的 CSV 模板直接下载
func (h *ImportHandler) DownloadTemplate(c *gin.Context) {
	csvData := service.GetStandardTemplateCSV()
	c.Header("Content-Disposition", "attachment; filename=\"standard_question_template.csv\"")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, csvData)
}

