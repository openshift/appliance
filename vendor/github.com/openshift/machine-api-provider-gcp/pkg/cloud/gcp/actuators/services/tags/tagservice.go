package tagservice

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2/google"
	tags "google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// TagService is a pass through wrapper for google.golang.org/api/cloudresourcemanager/v3
// to enable tests to mock this struct and control behavior.
type TagService interface {
	GetNamespacedName(context.Context, string) (*tags.TagValue, error)
}

// tagService implements TagService interface.
type tagService struct {
	tagValuesService *tags.TagValuesService
}

// BuilderFuncType is function type for building GCP tag client.
type BuilderFuncType func(ctx context.Context, serviceAccountJSON string) (TagService, error)

// NewTagService return a new tagService.
func NewTagService(ctx context.Context, serviceAccountJSON string) (TagService, error) {
	// Parse the credential type to allow restricting which credential types are
	// accepted from external sources. In this case, there are no restrictions
	// so we simply pass the type through.
	var f struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(serviceAccountJSON), &f); err != nil {
		return nil, fmt.Errorf("failed to parse credentials JSON: %w", err)
	}

	creds, err := google.CredentialsFromJSONWithType(ctx, []byte(serviceAccountJSON), google.CredentialsType(f.Type), compute.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("could not parse credentials: %w", err)
	}
	ud, err := creds.GetUniverseDomain()
	if err != nil {
		return nil, fmt.Errorf("could not get universe domain: %w", err)
	}
	service, err := tags.NewService(ctx, option.WithAuthCredentialsJSON(option.CredentialsType(f.Type), []byte(serviceAccountJSON)), option.WithUniverseDomain(ud))
	if err != nil {
		return nil, fmt.Errorf("could not create new tag service: %w", err)
	}

	return &tagService{
		tagValuesService: tags.NewTagValuesService(service),
	}, nil
}

// GetNamespacedName returns the tag's metadata fetched using its namespaced name.
func (t *tagService) GetNamespacedName(ctx context.Context, namespacedName string) (*tags.TagValue, error) {
	return t.tagValuesService.GetNamespaced().
		Context(ctx).
		Name(namespacedName).
		Do()
}
