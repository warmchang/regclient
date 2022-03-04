package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/internal/wraperr"
	"github.com/regclient/regclient/types"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/platform"
)

const (
	// MediaTypeOCI1Manifest OCI v1 manifest media type
	MediaTypeOCI1Manifest = types.MediaTypeOCI1Manifest
	// MediaTypeOCI1ManifestList OCI v1 manifest list media type
	MediaTypeOCI1ManifestList = types.MediaTypeOCI1ManifestList
)

type oci1Manifest struct {
	common
	v1.Manifest
}
type oci1Index struct {
	common
	v1.Index
}

func (m *oci1Manifest) GetConfig() (types.Descriptor, error) {
	return m.Config, nil
}
func (m *oci1Manifest) GetConfigDigest() (digest.Digest, error) {
	return m.Config.Digest, nil
}
func (m *oci1Index) GetConfig() (types.Descriptor, error) {
	return types.Descriptor{}, wraperr.New(fmt.Errorf("config digest not available for media type %s", m.desc.MediaType), types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetConfigDigest() (digest.Digest, error) {
	return "", wraperr.New(fmt.Errorf("config digest not available for media type %s", m.desc.MediaType), types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) GetManifestList() ([]types.Descriptor, error) {
	return []types.Descriptor{}, wraperr.New(fmt.Errorf("platform descriptor list not available for media type %s", m.desc.MediaType), types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetManifestList() ([]types.Descriptor, error) {
	return m.Manifests, nil
}

func (m *oci1Manifest) GetLayers() ([]types.Descriptor, error) {
	return m.Layers, nil
}
func (m *oci1Index) GetLayers() ([]types.Descriptor, error) {
	return []types.Descriptor{}, wraperr.New(fmt.Errorf("layers are not available for media type %s", m.desc.MediaType), types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) GetOrig() interface{} {
	return m.Manifest
}
func (m *oci1Index) GetOrig() interface{} {
	return m.Index
}

func (m *oci1Manifest) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	return nil, wraperr.New(fmt.Errorf("platform lookup not available for media type %s", m.desc.MediaType), types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	dl, err := m.GetManifestList()
	if err != nil {
		return nil, err
	}
	return getPlatformDesc(p, dl)
}

func (m *oci1Manifest) GetPlatformList() ([]*platform.Platform, error) {
	return nil, wraperr.New(fmt.Errorf("platform list not available for media type %s", m.desc.MediaType), types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetPlatformList() ([]*platform.Platform, error) {
	dl, err := m.GetManifestList()
	if err != nil {
		return nil, err
	}
	return getPlatformList(dl)
}

func (m *oci1Manifest) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, wraperr.New(fmt.Errorf("Manifest unavailable, perform a ManifestGet first"), types.ErrUnavailable)
	}

	if len(m.rawBody) > 0 {
		return m.rawBody, nil
	}

	return json.Marshal((m.Manifest))
}
func (m *oci1Index) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, wraperr.New(fmt.Errorf("Manifest unavailable, perform a ManifestGet first"), types.ErrUnavailable)
	}

	if len(m.rawBody) > 0 {
		return m.rawBody, nil
	}

	return json.Marshal((m.Index))
}

func (m *oci1Manifest) MarshalPretty() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(m.Manifest)
	return buf.Bytes(), nil
}
func (m *oci1Index) MarshalPretty() ([]byte, error) {
	if m == nil {
		return []byte{}, nil
	}
	buf := &bytes.Buffer{}
	tw := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	if m.r.Reference != "" {
		fmt.Fprintf(tw, "Name:\t%s\n", m.r.Reference)
	}
	fmt.Fprintf(tw, "MediaType:\t%s\n", m.desc.MediaType)
	fmt.Fprintf(tw, "Digest:\t%s\n", m.desc.Digest.String())
	if m.Annotations != nil && len(m.Annotations) > 0 {
		fmt.Fprintf(tw, "Annotations:\t\n")
		for name, val := range m.Annotations {
			fmt.Fprintf(tw, "  %s:\t%s\n", name, val)
		}
	}
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Manifests:\t\n")
	for _, d := range m.Manifests {
		fmt.Fprintf(tw, "\t\n")
		dRef := m.r
		if dRef.Reference != "" {
			dRef.Digest = d.Digest.String()
			fmt.Fprintf(tw, "  Name:\t%s\n", dRef.CommonName())
		} else {
			fmt.Fprintf(tw, "  Digest:\t%s\n", string(d.Digest))
		}
		fmt.Fprintf(tw, "  MediaType:\t%s\n", d.MediaType)
		if d.Platform != nil {
			if p := d.Platform; p.OS != "" {
				fmt.Fprintf(tw, "  Platform:\t%s\n", *p)
				if p.OSVersion != "" {
					fmt.Fprintf(tw, "  OSVersion:\t%s\n", p.OSVersion)
				}
				if len(p.OSFeatures) > 0 {
					fmt.Fprintf(tw, "  OSFeatures:\t%s\n", strings.Join(p.OSFeatures, ", "))
				}
			}
		}
		if len(d.URLs) > 0 {
			fmt.Fprintf(tw, "  URLs:\t%s\n", strings.Join(d.URLs, ", "))
		}
		if d.Annotations != nil {
			fmt.Fprintf(tw, "  Annotations:\t\n")
			for k, v := range d.Annotations {
				fmt.Fprintf(tw, "    %s:\t%s\n", k, v)
			}
		}
	}
	tw.Flush()
	return buf.Bytes(), nil
}

func (m *oci1Manifest) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(v1.Manifest)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeOCI1Manifest {
		// TODO: error?
		orig.MediaType = types.MediaTypeOCI1Manifest
	}
	mj, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	m.manifSet = true
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeOCI1Manifest,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	m.Manifest = orig

	return nil
}

func (m *oci1Index) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(v1.Index)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeOCI1ManifestList {
		// TODO: error?
		orig.MediaType = types.MediaTypeOCI1ManifestList
	}
	mj, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	m.manifSet = true
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeOCI1ManifestList,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	m.Index = orig

	return nil
}