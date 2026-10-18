package client

import (
	"net/url"
	"testing"
)

// The API rejects self_service_recommended, even false, unless Self Service is on.
func TestCustomAppFormSelfServiceRecommended(t *testing.T) {
	off, _ := url.ParseQuery(string(customAppForm(CustomApp{Name: "a"})))
	if off.Has("self_service_recommended") {
		t.Errorf("self_service_recommended sent with Self Service off: %v", off)
	}

	on, _ := url.ParseQuery(string(customAppForm(CustomApp{Name: "a", ShowInSelfService: true, SelfServiceRecommended: true})))
	if on.Get("self_service_recommended") != "true" {
		t.Errorf("self_service_recommended = %q with Self Service on, want true", on.Get("self_service_recommended"))
	}
}
