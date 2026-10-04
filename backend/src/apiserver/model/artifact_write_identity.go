package model

type ArtifactWriteIdentity struct {
	Identity        string `gorm:"column:Identity;not null;primaryKey;type:varchar(191);"`
	Namespace       string `gorm:"column:Namespace;not null;type:varchar(191);index;"`
	RunUUID         string `gorm:"column:RunUUID;not null;type:varchar(191);index;"`
	TaskID          string `gorm:"column:TaskID;not null;type:varchar(191);"`
	RetryGeneration int64  `gorm:"column:RetryGeneration;not null;"`
	ProducerKey     string `gorm:"column:ProducerKey;not null;type:varchar(191);"`
	IterationIndex  int64  `gorm:"column:IterationIndex;not null;"`
	PayloadHash     string `gorm:"column:PayloadHash;not null;type:varchar(64);"`
	ArtifactID      string `gorm:"column:ArtifactID;not null;type:varchar(191);index;"`
}

func (ArtifactWriteIdentity) TableName() string {
	return "artifact_write_identities"
}
