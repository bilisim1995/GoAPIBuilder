package handlers

import (
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "regexp"
        "strings"
        "time"

        "go.mongodb.org/mongo-driver/bson"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// GetKurumDuyuru scrapes announcements from institution's website
func GetKurumDuyuru(w http.ResponseWriter, r *http.Request) {
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Get query parameters
        kurumID := r.URL.Query().Get("kurum_id")
        if kurumID == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "kurum_id parameter is required")
                return
        }

        collection := config.GetKurumDuyuruCollection(mongoClient)

        // Find duyuru_linki by kurum_id
        var kurumDuyuru models.KurumDuyuru
        filter := bson.M{"kurum_id": kurumID}
        
        if err := collection.FindOne(ctx, filter).Decode(&kurumDuyuru); err != nil {
                utils.SendErrorResponse(w, http.StatusNotFound, "Kurum duyuru linki bulunamadı: "+err.Error())
                return
        }

        if kurumDuyuru.DuyuruLinki == "" {
                utils.SendErrorResponse(w, http.StatusNotFound, "Kurum için duyuru linki tanımlanmamış")
                return
        }

        // Scrape the website
        duyurular, err := scrapeYargitayDuyuru(kurumDuyuru.DuyuruLinki)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Duyuru sayfası çekilemedi: "+err.Error())
                return
        }

        // Prepare response
        response := models.APIResponse{
                Success: true,
                Data:    duyurular,
                Count:   len(duyurular),
                Message: "Kurum duyuruları başarıyla çekildi",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// scrapeYargitayDuyuru scrapes announcements from Yargıtay website
func scrapeYargitayDuyuru(url string) ([]models.DuyuruItem, error) {
        // Create HTTP client with timeout
        client := &http.Client{
                Timeout: 15 * time.Second,
        }

        // Make GET request
        resp, err := client.Get(url)
        if err != nil {
                return nil, fmt.Errorf("HTTP request failed: %v", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
        }

        // Read response body
        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return nil, fmt.Errorf("Failed to read response: %v", err)
        }

        // Parse HTML with regex patterns (simpler approach)
        duyurular := extractDuyurularWithRegex(string(body), url)

        // Limit to 5 results
        if len(duyurular) > 5 {
                duyurular = duyurular[:5]
        }

        return duyurular, nil
}

// extractDuyurularWithRegex extracts announcements using regex patterns
func extractDuyurularWithRegex(htmlContent, baseURL string) []models.DuyuruItem {
        var duyurular []models.DuyuruItem
        seenLinks := make(map[string]bool) // Deduplication

        // Primary regex: Yargıtay item links + keyword-based links with nested HTML support
        linkPattern := regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*(?:/item/\d+/[^"']*|(?:duyuru|haber|news|announcement)[^"']*))["'][^>]*>([\s\S]*?)</a>`)
        matches := linkPattern.FindAllStringSubmatch(htmlContent, -1)
        
        for _, match := range matches {
                if len(match) >= 3 {
                        href := strings.TrimSpace(match[1])
                        innerHTML := strings.TrimSpace(match[2])
                        
                        // Clean title from HTML entities and extract text
                        title := cleanHTML(innerHTML)
                        
                        // Normalize link for deduplication
                        normalizedLink := makeAbsoluteURL(href, baseURL)
                        
                        if len(title) > 10 && href != "" && !seenLinks[normalizedLink] {
                                seenLinks[normalizedLink] = true
                                duyuru := models.DuyuruItem{
                                        Baslik: title,
                                        Link:   normalizedLink,
                                        Tarih:  extractDateFromHTML(htmlContent, title),
                                }
                                duyurular = append(duyurular, duyuru)
                        }
                }
        }
        
        // Secondary pass: if we have fewer than 5 items, try general item links
        if len(duyurular) < 5 {
                itemLinkPattern := regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*/item/\d+/[^"']*)["'][^>]*>([\s\S]*?)</a>`)
                itemMatches := itemLinkPattern.FindAllStringSubmatch(htmlContent, -1)
                
                for _, match := range itemMatches {
                        if len(duyurular) >= 5 {
                                break
                        }
                        
                        if len(match) >= 3 {
                                href := strings.TrimSpace(match[1])
                                innerHTML := strings.TrimSpace(match[2])
                                title := cleanHTML(innerHTML)
                                
                                normalizedLink := makeAbsoluteURL(href, baseURL)
                                
                                if len(title) > 10 && !seenLinks[normalizedLink] && !isNavigationLink(title) {
                                        seenLinks[normalizedLink] = true
                                        duyuru := models.DuyuruItem{
                                                Baslik: title,
                                                Link:   normalizedLink,
                                                Tarih:  extractDateFromHTML(htmlContent, title),
                                        }
                                        duyurular = append(duyurular, duyuru)
                                }
                        }
                }
        }
        
        // Final fallback: general links if still not enough
        if len(duyurular) < 5 {
                generalLinkPattern := regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*)["'][^>]*>([\s\S]{15,}?)</a>`)
                generalMatches := generalLinkPattern.FindAllStringSubmatch(htmlContent, -1)
                
                for _, match := range generalMatches {
                        if len(duyurular) >= 5 {
                                break
                        }
                        
                        if len(match) >= 3 {
                                href := strings.TrimSpace(match[1])
                                innerHTML := strings.TrimSpace(match[2])
                                title := cleanHTML(innerHTML)
                                
                                normalizedLink := makeAbsoluteURL(href, baseURL)
                                
                                if len(title) > 15 && !seenLinks[normalizedLink] && !isNavigationLink(title) {
                                        seenLinks[normalizedLink] = true
                                        duyuru := models.DuyuruItem{
                                                Baslik: title,
                                                Link:   normalizedLink,
                                                Tarih:  extractDateFromHTML(htmlContent, title),
                                        }
                                        duyurular = append(duyurular, duyuru)
                                }
                        }
                        }
        }
        
        return duyurular
}

// cleanHTML removes HTML entities and cleans text
func cleanHTML(text string) string {
        // Remove HTML tags
        text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
        
        // Replace common HTML entities
        text = strings.ReplaceAll(text, "&nbsp;", " ")
        text = strings.ReplaceAll(text, "&amp;", "&")
        text = strings.ReplaceAll(text, "&quot;", "\"")
        text = strings.ReplaceAll(text, "&lt;", "<")
        text = strings.ReplaceAll(text, "&gt;", ">")
        
        // Clean extra spaces
        text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
        
        return strings.TrimSpace(text)
}

// isNavigationLink checks if the text looks like a navigation item
func isNavigationLink(text string) bool {
        navKeywords := []string{
                "ana sayfa", "anasayfa", "home", "menü", "menu",
                "hakkımızda", "iletişim", "contact", "about",
                "giriş", "login", "kayıt", "register", "çıkış", "logout",
                "ara", "search", "site haritası", "sitemap",
        }
        
        lowerText := strings.ToLower(text)
        for _, keyword := range navKeywords {
                if strings.Contains(lowerText, keyword) {
                        return true
                }
        }
        
        // Check for very short texts (likely navigation)
        return len(strings.TrimSpace(text)) < 15
}

// extractDateFromHTML tries to find date near the announcement title
func extractDateFromHTML(htmlContent, title string) string {
        // Look for date patterns near the title
        titleIndex := strings.Index(htmlContent, title)
        if titleIndex == -1 {
                return time.Now().Format("02.01.2006")
        }
        
        // Search in surrounding text (500 characters before and after)
        start := titleIndex - 500
        if start < 0 {
                start = 0
        }
        end := titleIndex + len(title) + 500
        if end > len(htmlContent) {
                end = len(htmlContent)
        }
        
        surroundingText := htmlContent[start:end]
        if date := extractDateFromText(surroundingText); date != "" {
                return date
        }
        
        return time.Now().Format("02.01.2006")
}

func makeAbsoluteURL(href, baseURL string) string {
        if strings.HasPrefix(href, "http") {
                return href
        }
        if strings.HasPrefix(href, "/") {
                // Extract base domain from baseURL
                if idx := strings.Index(baseURL[8:], "/"); idx > 0 {
                        return baseURL[:8+idx] + href
                }
                return baseURL + href
        }
        return baseURL + "/" + href
}

func extractDateFromText(text string) string {
        // Common Turkish date patterns
        datePatterns := []string{
                `\d{1,2}[./]\d{1,2}[./]\d{4}`,        // 01.01.2024 or 01/01/2024
                `\d{1,2}[./]\d{1,2}[./]\d{2}`,         // 01.01.24
                `\d{4}[.-]\d{1,2}[.-]\d{1,2}`,         // 2024-01-01
        }
        
        for _, pattern := range datePatterns {
                re := regexp.MustCompile(pattern)
                if match := re.FindString(text); match != "" {
                        return match
                }
        }
        
        return ""
}

