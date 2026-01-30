package onboarding

import (
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// Clients holds the clients created during onboarding
type Clients struct {
	WAClient   *whatsapp.Client
	GCalClient *gcal.Client
	MsgChan    <-chan source.Message
}
