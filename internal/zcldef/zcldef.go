package zcldef

import (
	"log"
)

type ZCLDefService interface {
	GetById(clusterId uint16) ClusterDefinition
}

type zclDefService struct {
	zclDefMap *map[uint16]ClusterDefinition
}

func (zd *zclDefService) GetById(clusterId uint16) ClusterDefinition {
	return (*zd.zclDefMap)[clusterId]
}

func New(filename string) ZCLDefService {
	zclDef := loadFromFile(filename)
	if zclDef == nil {
		log.Fatalf("Failed to load ZCL definition from file: %v", filename)
	}

	return &zclDefService{
		zclDefMap: zclDef,
	}
}
