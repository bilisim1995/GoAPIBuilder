package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InstitutionModel represents the institutions collection structure
type InstitutionModel struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	KurumAdi    string             `bson:"kurum_adi" json:"kurum_adi"`
	KurumLogo   string             `bson:"kurum_logo" json:"kurum_logo"`
	Aciklama    string             `bson:"aciklama" json:"aciklama"`
	Website     string             `bson:"website" json:"website"`
	Aktif       bool               `bson:"aktif" json:"aktif"`
	CreatedAt   string             `bson:"created_at" json:"created_at"`
	UpdatedAt   string             `bson:"updated_at" json:"updated_at"`
}

// InstitutionCreateRequest represents request for creating institution
type InstitutionCreateRequest struct {
	KurumAdi  string `json:"kurum_adi" binding:"required"`
	KurumLogo string `json:"kurum_logo"`
	Aciklama  string `json:"aciklama"`
	Website   string `json:"website"`
}

// InstitutionUpdateRequest represents request for updating institution
type InstitutionUpdateRequest struct {
	KurumAdi  string `json:"kurum_adi"`
	KurumLogo string `json:"kurum_logo"`
	Aciklama  string `json:"aciklama"`
	Website   string `json:"website"`
	Aktif     *bool  `json:"aktif"`
}

// InstitutionListResponse represents simplified institution for listing
type InstitutionListResponse struct {
	ID        string `json:"id"`
	KurumAdi  string `json:"kurum_adi"`
	KurumLogo string `json:"kurum_logo"`
	Aktif     bool   `json:"aktif"`
}