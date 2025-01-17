// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/mattermost/mattermost-server/v6/store"
)

func (a *App) GetComplianceReports(page, perPage int) (model.Compliances, *model.AppError) {
	if !*a.Config().ComplianceSettings.Enable {
		return nil, model.NewAppError("GetComplianceReports", "ent.compliance.licence_disable.app_error", nil, "", http.StatusNotImplemented)
	}

	compliances, err := a.Srv().Store.Compliance().GetAll(page*perPage, perPage)
	if err != nil {
		return nil, model.NewAppError("GetComplianceReports", "app.compliance.get.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return compliances, nil
}

func (a *App) SaveComplianceReport(job *model.Compliance) (*model.Compliance, *model.AppError) {
	if !*a.Config().ComplianceSettings.Enable || a.Compliance() == nil {
		return nil, model.NewAppError("saveComplianceReport", "ent.compliance.licence_disable.app_error", nil, "", http.StatusNotImplemented)
	}

	job.Type = model.ComplianceTypeAdhoc

	job, err := a.Srv().Store.Compliance().Save(job)
	if err != nil {
		var appErr *model.AppError
		switch {
		case errors.As(err, &appErr):
			return nil, appErr
		default:
			return nil, model.NewAppError("SaveComplianceReport", "app.compliance.save.saving.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	jCopy := job.DeepCopy()
	a.Srv().Go(func() {
		err := a.Compliance().RunComplianceJob(jCopy)
		if err != nil {
			mlog.Warn("Error running compliance job", mlog.Err(err))
		}
	})

	return job, nil
}

func (a *App) GetComplianceReport(reportId string) (*model.Compliance, *model.AppError) {
	if !*a.Config().ComplianceSettings.Enable || a.Compliance() == nil {
		return nil, model.NewAppError("downloadComplianceReport", "ent.compliance.licence_disable.app_error", nil, "", http.StatusNotImplemented)
	}

	compliance, err := a.Srv().Store.Compliance().Get(reportId)
	if err != nil {
		var nfErr *store.ErrNotFound
		switch {
		case errors.As(err, &nfErr):
			return nil, model.NewAppError("GetComplianceReport", "app.compliance.get.finding.app_error", nil, nfErr.Error(), http.StatusNotFound)
		default:
			return nil, model.NewAppError("GetComplianceReport", "app.compliance.get.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	return compliance, nil
}

func (a *App) GetComplianceFile(job *model.Compliance) ([]byte, *model.AppError) {
	f, err := ioutil.ReadFile(*a.Config().ComplianceSettings.Directory + "compliance/" + job.JobName() + ".zip")
	if err != nil {
		return nil, model.NewAppError("readFile", "api.file.read_file.reading_local.app_error", nil, err.Error(), http.StatusNotImplemented)
	}
	return f, nil
}
