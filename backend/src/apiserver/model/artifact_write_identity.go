package model

type ArtifactWriteIdentity struct {
	Identity    string `gorm:"column:Identity;not null;primaryKey;type:varchar(191);"`
	Namespace   string `gorm:"column:Namespace;not null;type:varchar(191);index;"`
	RunUUID     string `gorm:"column:RunUUID;not null;type:varchar(191);index;"`
	TaskID      string `gorm:"column:TaskID;not null;type:varchar(191);"`
	OperationID string `gorm:"column:OperationID;not null;type:varchar(191);"`
	PayloadHash string `gorm:"column:PayloadHash;not null;type:varchar(64);"`
	ArtifactID  string `gorm:"column:ArtifactID;not null;type:varchar(191);index;"`
}

func (ArtifactWriteIdentity) TableName() string {
	return "artifact_write_identities"
}
