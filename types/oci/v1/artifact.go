package v1

import (
	"github.com/regclient/regclient/types"
)

// ArtifactManifest defines an OCI Artifact
type ArtifactManifest struct {
	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType"`

	// ArtifactType is the media type of the artifact this schema refers to.
	ArtifactType string `json:"artifactType"`

	// Blobs is a collection of blobs referenced by this manifest.
	Blobs []types.Descriptor `json:"blobs,omitempty"`

	// Refers indicates this manifest references another manifest
	// TODO: deprecated, delete this from future releases
	Refers *types.Descriptor `json:"refers,omitempty"`

	// Subject is an optional link from the image manifest to another manifest forming an association between the image manifest and the other manifest.
	Subject *types.Descriptor `json:"subject,omitempty"`

	// Annotations contains arbitrary metadata for the artifact manifest.
	Annotations map[string]string `json:"annotations,omitempty"`
}
