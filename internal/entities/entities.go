package entities

type ContentMapping struct {
	Cid  string `json:"cid" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type DeployProof struct {
	Signature string `json:"signature" validate:"required"`
	Address   string `json:"address" validate:"required"`
	ID        string `json:"id" validate:"required"`
	Timestamp int64  `json:"timestamp" validate:"required"`
}

type Deploy struct {
	Required  []ContentMapping `json:"required" validate:"required,gt=0"`
	Positions []string         `json:"positions" validate:"required,gt=0"`
	Mappings  string           `json:"mappings" validate:"required"`
	Timestamp int64            `json:"timestamp" validate:"required"`
}

func (d *Deploy) UniquePositions() []string {
	parcels := map[string]bool{}
	for _, p := range d.Positions {
		parcels[p] = true
	}
	unique := make([]string, len(parcels))
	i := 0
	for k := range parcels {
		unique[i] = k
		i++
	}
	return unique
}
