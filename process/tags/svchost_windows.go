package tags

import (
	"fmt"
	"strings"

	"github.com/khulnasoft-lab/portmaster/process"
	"github.com/khulnasoft-lab/portmaster/profile"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils/osdetail"
)

func init() {
	err := process.RegisterTagHandler(new(SVCHostTagHandler))
	if err != nil {
		panic(err)
	}
}

const (
	svchostName   = "Service Host"
	svchostTagKey = "svchost"
)

// SVCHostTagHandler handles svchost processes on Windows.
type SVCHostTagHandler struct{}

// Name returns the tag handler name.
func (h *SVCHostTagHandler) Name() string {
	return svchostName
}

// TagDescriptions returns a list of all possible tags and their description
// of this handler.
func (h *SVCHostTagHandler) TagDescriptions() []process.TagDescription {
	return []process.TagDescription{
		process.TagDescription{
			ID:          svchostTagKey,
			Name:        "SvcHost Service Name",
			Description: "Name of a service running in svchost.exe as reported by Windows.",
		},
	}
}

// TagKeys returns a list of all possible tag keys of this handler.
func (h *SVCHostTagHandler) TagKeys() []string {
	return []string{svchostTagKey}
}

// AddTags adds tags to the given process.
func (h *SVCHostTagHandler) AddTags(p *process.Process) {
	// Check for svchost.exe.
	if p.ExecName != "svchost.exe" {
		return
	}

	// Get services of svchost instance.
	svcNames, err := osdetail.GetServiceNames(int32(p.Pid))
	switch err {
	case nil:
		// Append service names to process name.
		p.Name += fmt.Sprintf(" (%s)", strings.Join(svcNames, ", "))
		// Add services as tags.
		for _, svcName := range svcNames {
			// Remove tags from service names, such as "CDPUserSvc_1bf5729".
			svcName, _, _ := strings.Cut(svcName, "_")
			// Add service as tag.
			p.Tags = append(p.Tags, profile.Tag{
				Key:   svchostTagKey,
				Value: svcName,
			})
		}
	case osdetail.ErrServiceNotFound:
		log.Tracef("process/tags: failed to get service name for svchost.exe (pid %d): %s", p.Pid, err)
	default:
		log.Warningf("process/tags: failed to get service name for svchost.exe (pid %d): %s", p.Pid, err)
	}
}

// CreateProfile creates a profile based on the tags of the process.
// Returns nil to skip.
func (h *SVCHostTagHandler) CreateProfile(p *process.Process) *profile.Profile {
	if tag, ok := p.GetTag(svchostTagKey); ok {
		return profile.New(&profile.Profile{
			Source:              profile.SourceLocal,
			Name:                "Windows Service: " + osdetail.GenerateBinaryNameFromPath(tag.Value),
			Icon:                `C:\Windows\System32\@WLOGO_48x48.png`,
			IconType:            profile.IconTypeFile,
			UsePresentationPath: false,
			Fingerprints: []profile.Fingerprint{
				profile.Fingerprint{
					Type:      profile.FingerprintTypeTagID,
					Key:       tag.Key,
					Operation: profile.FingerprintOperationEqualsID,
					Value:     tag.Value, // Value of svchostTagKey.
				},
			},
		})
	}

	return nil
}
