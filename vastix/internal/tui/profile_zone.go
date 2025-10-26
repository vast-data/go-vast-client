package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"vastix/internal/msg_types"

	"vastix/internal/database"
	"vastix/internal/logging"
	log "vastix/internal/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	vast_client "github.com/vast-data/go-vast-client"
	"go.uber.org/zap"
)

// ProfileZone represents the profile information zone
type ProfileZone struct {
	ready          bool // Indicates if the profile zone is ready to be displayed
	width, height  int
	profileName    string
	availableSpace string
	usedSpace      string
	freeSpace      string
	userName       string
	tenant         string
	token          string
	db             *database.Service
}

// NewProfileZone creates a new profile zone with logging support
func NewProfileZone(db *database.Service) *ProfileZone {
	profile := &ProfileZone{
		availableSpace: "n/a",
		usedSpace:      "n/a",
		freeSpace:      "n/a",
		db:             db,
		ready:          true,
	}

	activeProfile, err := db.GetActiveProfile()
	if err != nil {
		log.Error("Error getting active profile", zap.Error(err))
	} else if activeProfile != nil {
		profile.profileName = activeProfile.ProfileName()
		profile.tenant = activeProfile.Tenant
		profile.userName = activeProfile.Username
		profile.token = activeProfile.Token
	}

	return profile
}

func (p *ProfileZone) Init() {}

func (p *ProfileZone) SetData() tea.Msg {
	auxlog := logging.GetAuxLogger()
	auxlog.Println("ProfileZone.SetData: triggered")

	// Try to get the active profile
	activeProfile, err := p.db.GetActiveProfile()

	if err != nil {
		log.Error("Failed to get active profile", zap.Error(err))
		auxlog.Printf("ProfileZone.SetData: failed: %v", err)
		return nil
	} else if activeProfile != nil {
		// Set basic profile info immediately (non-blocking)
		p.profileName = activeProfile.ProfileName()
		p.tenant = activeProfile.Tenant
		p.userName = activeProfile.Username
		p.token = activeProfile.Token
		auxlog.Printf("ProfileZone.SetData: loaded profile: %s", activeProfile.ProfileName())

		// Return async command to fetch space metrics
		metrics, err := p.fetchSpaceMetrics(activeProfile)
		if err != nil {
			log.Error("Failed to fetch space metrics", zap.Error(err))
			auxlog.Printf("ProfileZone.SetData() metrics fetch failed: %v", err)
			return msg_types.ProfileDataMsg{
				AvailableSpace: "n/a",
				UsedSpace:      "n/a",
				FreeSpace:      "n/a",
			}
		}
		return *metrics
	} else {
		// No active profile found
		log.Debug("No active profile found")
		auxlog.Println("ProfileZone.SetData: no active profile found")
		return nil
	}
}

func (p *ProfileZone) fetchSpaceMetrics(activeProfile *database.Profile) (*msg_types.ProfileDataMsg, error) {
	rest, err := activeProfile.RestClientFromProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST client from profile: %w", err)
	}

	metrics := []string{
		"Capacity,drr",
		"Capacity,logical_space",
		"Capacity,logical_space_in_use",
		"Capacity,physical_space",
		"Capacity,physical_space_in_use",
	}

	// Build params for the ad_hoc_query extra method
	params := vast_client.Params{
		"object_type": "cluster",
		"time_frame":  "5m",
		"prop_list":   metrics,
	}

	// Use the Monitors resource's MonitorAdHocQuery_GET extra method
	res, err := rest.Monitors.MonitorAdHocQuery_GET(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get ad_hoc_query data: %w", err)
	}

	// Parse response
	var parsed struct {
		Data     [][]interface{} `json:"data"`
		PropList []string        `json:"prop_list"`
	}
	raw, _ := json.Marshal(res)
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse ad_hoc_query data: %w", err)
	}

	if len(parsed.Data) == 0 {
		log.Debug("No cluster metrics data available")
		return &msg_types.ProfileDataMsg{
			AvailableSpace: "n/a",
			UsedSpace:      "n/a",
			FreeSpace:      "n/a",
		}, nil
	}

	last := parsed.Data[len(parsed.Data)-1]
	metricsMap := map[string]interface{}{}
	for i, name := range parsed.PropList {
		refined := name
		if idx := findComma(name); idx != -1 {
			refined = name[idx+1:]
		}
		if i < len(last) {
			metricsMap[refined] = last[i]
		}
	}

	const GiB = 1 << 30
	total := float64OrZero(metricsMap["logical_space"]) / GiB
	used := float64OrZero(metricsMap["logical_space_in_use"]) / GiB
	free := total - used

	return &msg_types.ProfileDataMsg{
		AvailableSpace: fmt.Sprintf("%.2f GiB", total),
		UsedSpace:      fmt.Sprintf("%.2f GiB", used),
		FreeSpace:      fmt.Sprintf("%.2f GiB", free),
	}, nil

}

// UpdateData updates the profile zone with space metrics data
func (p *ProfileZone) UpdateData(data msg_types.ProfileDataMsg) {
	p.availableSpace = data.AvailableSpace
	p.usedSpace = data.UsedSpace
	p.freeSpace = data.FreeSpace
	log.Debug("Profile zone data updated",
		zap.String("available", p.availableSpace),
		zap.String("used", p.usedSpace),
		zap.String("free", p.freeSpace))
}

// SetSize sets the dimensions of the profile zone
func (p *ProfileZone) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// View renders the profile zone
func (p *ProfileZone) View() string {
	if p.width == 0 || p.profileName == "" {
		return ""
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(LightGrey).
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(White).
		Bold(true)

	lines := []string{
		fmt.Sprintf("%s %s",
			keyStyle.Render("Profile:"),
			valueStyle.Render(p.profileName)),
	}
	if p.userName != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			keyStyle.Render("User:"),
			valueStyle.Render(p.userName)))
	} else if p.token != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			keyStyle.Render("Token:"),
			valueStyle.Render("[redacted]")))
	}
	if p.tenant != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			keyStyle.Render("Tenant:"),
			valueStyle.Render(p.tenant)))
	}
	if p.freeSpace != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			keyStyle.Render("Free:"),
			valueStyle.Render(p.freeSpace)))
	}
	if p.availableSpace != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			keyStyle.Render("Available:"),
			valueStyle.Render(p.availableSpace)))
	}
	if p.usedSpace != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			keyStyle.Render("Used:"),
			valueStyle.Render(p.usedSpace)))
	}
	return strings.Join(lines, "\n")
}

// Ready returns whether the profile zone is ready to be displayed
func (p *ProfileZone) Ready() bool {
	return p.ready
}

func findComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}

func float64OrZero(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		return 0
	}
}
