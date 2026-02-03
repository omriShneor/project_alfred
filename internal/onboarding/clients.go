package onboarding

import (
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// Clients holds the clients created during onboarding
// Note: GCal client is created per-user after authentication, not at startup
type Clients struct {
	WAClient *whatsapp.Client
	MsgChan  <-chan source.Message
}
