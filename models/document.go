package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentMetadata represents the metadata collection structure
type DocumentMetadata struct {
	ID                 primitive.ObjectID `bson:"_id" json:"id"`
	PdfAdi            string             `bson:"pdf_adi" json:"pdf_adi"`
	KurumAdi          string             `bson:"kurum_adi" json:"kurum_adi"`
	BelgeTuru         string             `bson:"belge_turu" json:"belge_turu"`
	BelgeDurumu       string             `bson:"belge_durumu" json:"belge_durumu"`
	BelgeYayinTarihi  string             `bson:"belge_yayin_tarihi" json:"belge_yayin_tarihi"`
	Etiketler         string             `bson:"etiketler" json:"etiketler"`
	AnahtarKelimeler  string             `bson:"anahtar_kelimeler" json:"anahtar_kelimeler"`
	Aciklama          string             `bson:"aciklama" json:"aciklama"`
	URLSlug           string             `bson:"url_slug" json:"url_slug"`
	Status            string             `bson:"status" json:"status"`
	SayfaSayisi       int32              `bson:"sayfa_sayisi" json:"sayfa_sayisi"`
	DosyaBoyutuMB     float64            `bson:"dosya_boyutu_mb" json:"dosya_boyutu_mb"`
	YuklemeTarihi     time.Time          `bson:"yukleme_tarihi" json:"yukleme_tarihi"`
	OlusturulmaTarihi string             `bson:"olusturulma_tarihi" json:"olusturulma_tarihi"`
	PdfURL            string             `bson:"pdf_url" json:"pdf_url"`
}

// DocumentContent represents the content collection structure
type DocumentContent struct {
	ID                primitive.ObjectID `bson:"_id" json:"id"`
	MetadataID        primitive.ObjectID `bson:"metadata_id" json:"metadata_id"`
	Icerik            string             `bson:"icerik" json:"icerik"`
	OlusturulmaTarihi string             `bson:"olusturulma_tarihi" json:"olusturulma_tarihi"`
}

// DocumentSummary represents a simplified document structure for listing
type DocumentSummary struct {
	ID               string `json:"id"`
	KurumAdi         string `json:"kurum_adi"`
	PdfAdi           string `json:"pdf_adi"`
	Etiketler        string `json:"etiketler"`
	BelgeYayinTarihi string `json:"belge_yayin_tarihi"`
	BelgeDurumu      string `json:"belge_durumu"`
	Aciklama         string `json:"aciklama"`
	URLSlug          string `json:"url_slug"`
}

// DocumentDetails represents the complete document with content
type DocumentDetails struct {
	Metadata DocumentMetadata `json:"metadata"`
	Content  DocumentContent  `json:"content"`
}

// Institution represents a unique institution
type Institution struct {
	KurumAdi string `json:"kurum_adi" bson:"_id"`
	Count    int32  `json:"count" bson:"count"`
}

// APIResponse represents a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Count   int         `json:"count,omitempty"`
}
