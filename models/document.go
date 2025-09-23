package models

import (
        "go.mongodb.org/mongo-driver/bson/primitive"
)

// Kurum represents institution data from kurumlar collection
type Kurum struct {
        ID            primitive.ObjectID `bson:"_id" json:"kurum_id"`
        KurumAdi      string             `bson:"kurum_adi" json:"kurum_adi"`
        KurumLogo     string             `bson:"kurum_logo" json:"kurum_logo"`
        KurumAciklama string             `bson:"aciklama" json:"kurum_aciklama"`
}

// DocumentMetadata represents the metadata collection structure
type DocumentMetadata struct {
        ID                 primitive.ObjectID `bson:"_id" json:"id"`
        PdfAdi            string             `bson:"pdf_adi" json:"pdf_adi"`
        KurumID           string             `bson:"kurum_id" json:"kurum_id"`
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
        YuklemeTarihi     string             `bson:"yukleme_tarihi" json:"yukleme_tarihi"`
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
        KurumLogo        string `json:"kurum_logo"`
        KurumAciklama    string `json:"kurum_aciklama"`
        PdfAdi           string `json:"pdf_adi"`
        BelgeTuru        string `json:"belge_turu"`
        Etiketler        string `json:"etiketler"`
        BelgeYayinTarihi string `json:"belge_yayin_tarihi"`
        BelgeDurumu      string `json:"belge_durumu"`
        Aciklama         string `json:"aciklama"`
        URLSlug          string `json:"url_slug"`
}

// DocumentDetails represents the complete document with content
type DocumentDetails struct {
        Metadata      DocumentMetadata `json:"metadata"`
        Content       DocumentContent  `json:"content"`
        KurumAdi      string           `json:"kurum_adi"`
        KurumLogo     string           `json:"kurum_logo"`
        KurumAciklama string           `json:"kurum_aciklama"`
}

// Institution represents a unique institution
type Institution struct {
        KurumID       string `json:"kurum_id" bson:"kurum_id"`
        KurumAdi      string `json:"kurum_adi" bson:"kurum_adi"`
        KurumLogo     string `json:"kurum_logo" bson:"kurum_logo"`
        KurumAciklama string `json:"kurum_aciklama" bson:"kurum_aciklama"`
        Count         int32  `json:"count" bson:"count"`
}

// KurumDuyuru represents institution announcement data from kurum_duyuru collection
type KurumDuyuru struct {
        ID       primitive.ObjectID `bson:"_id" json:"id"`
        KurumID  string             `bson:"kurum_id" json:"kurum_id"`
        Baslik   string             `bson:"baslik" json:"baslik"`
        Link     string             `bson:"link" json:"link"`
        Tarih    string             `bson:"tarih" json:"tarih"`
        Status   string             `bson:"status" json:"status"`
}

// APIResponse represents a standard API response structure
type APIResponse struct {
        Success bool        `json:"success"`
        Data    interface{} `json:"data,omitempty"`
        Error   string      `json:"error,omitempty"`
        Message string      `json:"message,omitempty"`
        Count   int         `json:"count,omitempty"`
}
