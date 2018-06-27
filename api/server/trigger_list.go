package server

import (
	"encoding/base64"
	"net/http"

	"fmt"
	"github.com/fnproject/fn/api/models"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleTriggerList(c *gin.Context) {
	ctx := c.Request.Context()

	filter := &models.TriggerFilter{}
	filter.Cursor, filter.PerPage = pageParams(c, true)

	filter.AppID = c.Query("app_id")

	if filter.AppID == "" {
		handleErrorResponse(c, models.ErrTriggerMissingAppID)
	}

	filter.FnID = c.Query("fn_id")
	filter.Name = c.Query("name")

	triggers, err := s.datastore.GetTriggers(ctx, filter)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	var nextCursor string
	if len(triggers) > 0 && len(triggers) == filter.PerPage {
		last := []byte(triggers[len(triggers)-1].ID)
		nextCursor = base64.RawURLEncoding.EncodeToString(last)
	}

	// Annotate the outbound triggers

	// this is fairly cludgy bit hard to do in datastore middleware confidently
	appCache := make(map[string]*models.App)
	newTriggers := make([]*models.Trigger, len(triggers))

	for idx, t := range triggers {
		app, ok := appCache[t.AppID]
		if !ok {
			gotApp, err := s.Datastore().GetAppByID(ctx, t.AppID)
			if err != nil {
				handleErrorResponse(c, fmt.Errorf("failed to get app for trigger %s", err))
				return
			}
			app = gotApp
			appCache[app.ID] = gotApp
		}

		newT, err := s.triggerAnnotator.AnnotateTrigger(c, app, t)
		if err != nil {
			handleErrorResponse(c, err)
			return
		}
		newTriggers[idx] = newT
	}

	c.JSON(http.StatusOK, triggerListResponse{
		NextCursor: nextCursor,
		Items:      newTriggers,
	})
}
