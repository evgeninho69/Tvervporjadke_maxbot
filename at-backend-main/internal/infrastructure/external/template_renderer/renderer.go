package template

import (
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"bytes"
	"context"
	"html/template"
	"os"
	"path/filepath"
)

type templateRenderer struct {
	loginEmailTemplate *template.Template
}

const opTemplateRendererNew = "infrastructure.external.TemplateRenderer.New"

func NewTemplateRenderer(resourcesPath string) (*templateRenderer, error) {
	loginTemplate, err := parseLoginHTML(resourcesPath)
	if err != nil {
		return nil, err
	}

	return &templateRenderer{loginEmailTemplate: loginTemplate}, nil
}

func parseLoginHTML(resourcesPath string) (*template.Template, error) {
	loginHTML, err := os.Open(filepath.Join(resourcesPath, "emails/login.html"))
	if err != nil {
		logs.Error(
			err.Error(),
			"main",
			opTemplateRendererNew,
		)

		return nil, apperror.ErrTemplateRenderer
	}

	buf := bytes.Buffer{}

	_, err = buf.ReadFrom(loginHTML)
	if err != nil {
		logs.Error(
			err.Error(),
			"main",
			opTemplateRendererNew,
		)

		return nil, apperror.ErrTemplateRenderer
	}

	loginTemplate, err := template.New("login").Parse(buf.String())
	if err != nil {
		logs.Error(
			err.Error(),
			"main",
			opTemplateRendererNew,
		)

		return nil, apperror.ErrTemplateRenderer
	}

	return loginTemplate, nil
}

const opTemplateRendererLoginEmail = "infrastructure.external.TemplateRenderer.RenderLoginEmail"

func (tr templateRenderer) RenderLoginEmailTemplate(ctx context.Context, values any) ([]byte, error) {
	buf := &bytes.Buffer{}

	if err := tr.loginEmailTemplate.Execute(buf, values); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opTemplateRendererLoginEmail,
		)

		return nil, apperror.ErrTemplateRenderer
	}

	return buf.Bytes(), nil
}