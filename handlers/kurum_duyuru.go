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
        "golang.org/x/net/html"

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

        // Parse HTML
        doc, err := html.Parse(strings.NewReader(string(body)))
        if err != nil {
                return nil, fmt.Errorf("HTML parse error: %v", err)
        }

        // Extract announcements
        var duyurular []models.DuyuruItem
        extractDuyurular(doc, &duyurular, url)

        // Limit to 5 results
        if len(duyurular) > 5 {
                duyurular = duyurular[:5]
        }

        return duyurular, nil
}

// extractDuyurular recursively searches HTML for announcement items
func extractDuyurular(n *html.Node, duyurular *[]models.DuyuruItem, baseURL string) {
        // Look for announcement list items or similar patterns
        if n.Type == html.ElementNode {
                // Check for common announcement patterns in Yargıtay
                if (n.Data == "div" || n.Data == "li" || n.Data == "tr") && hasAnnouncementClass(n) {
                        duyuru := extractSingleDuyuru(n, baseURL)
                        if duyuru.Baslik != "" && duyuru.Link != "" {
                                *duyurular = append(*duyurular, duyuru)
                        }
                }
                
                // Also look for simple link patterns
                if n.Data == "a" {
                        href := getAttr(n, "href")
                        if href != "" && strings.Contains(href, "duyuru") {
                                text := getTextContent(n)
                                if text != "" && len(text) > 10 { // Minimum meaningful title length
                                        duyuru := models.DuyuruItem{
                                                Baslik: strings.TrimSpace(text),
                                                Link:   makeAbsoluteURL(href, baseURL),
                                                Tarih:  extractDateFromText(text),
                                        }
                                        *duyurular = append(*duyurular, duyuru)
                                }
                        }
                }
        }

        // Recursively search child nodes
        for c := n.FirstChild; c != nil; c = c.NextSibling {
                extractDuyurular(c, duyurular, baseURL)
        }
}

// hasAnnouncementClass checks if node has announcement-related class or attributes
func hasAnnouncementClass(n *html.Node) bool {
        class := getAttr(n, "class")
        id := getAttr(n, "id")
        
        keywords := []string{"duyuru", "announcement", "news", "notice", "item", "list"}
        
        for _, keyword := range keywords {
                if strings.Contains(strings.ToLower(class), keyword) ||
                   strings.Contains(strings.ToLower(id), keyword) {
                        return true
                }
        }
        return false
}

// extractSingleDuyuru extracts title, link and date from a single announcement node
func extractSingleDuyuru(n *html.Node, baseURL string) models.DuyuruItem {
        var duyuru models.DuyuruItem
        
        // Find link and title
        linkNode := findFirstElement(n, "a")
        if linkNode != nil {
                duyuru.Link = makeAbsoluteURL(getAttr(linkNode, "href"), baseURL)
                duyuru.Baslik = strings.TrimSpace(getTextContent(linkNode))
        }
        
        // If no link found, try to get text content as title
        if duyuru.Baslik == "" {
                duyuru.Baslik = strings.TrimSpace(getTextContent(n))
        }
        
        // Extract date from the node or its siblings
        duyuru.Tarih = extractDateFromNode(n)
        
        return duyuru
}

// Helper functions
func getAttr(n *html.Node, key string) string {
        for _, attr := range n.Attr {
                if attr.Key == key {
                        return attr.Val
                }
        }
        return ""
}

func getTextContent(n *html.Node) string {
        if n.Type == html.TextNode {
                return n.Data
        }
        
        var text strings.Builder
        for c := n.FirstChild; c != nil; c = c.NextSibling {
                text.WriteString(getTextContent(c))
        }
        return text.String()
}

func findFirstElement(n *html.Node, tag string) *html.Node {
        if n.Type == html.ElementNode && n.Data == tag {
                return n
        }
        
        for c := n.FirstChild; c != nil; c = c.NextSibling {
                if result := findFirstElement(c, tag); result != nil {
                        return result
                }
        }
        return nil
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

func extractDateFromNode(n *html.Node) string {
        // Try to find date in the current node and siblings
        text := getTextContent(n)
        if date := extractDateFromText(text); date != "" {
                return date
        }
        
        // Try parent node
        if n.Parent != nil {
                parentText := getTextContent(n.Parent)
                if date := extractDateFromText(parentText); date != "" {
                        return date
                }
        }
        
        return time.Now().Format("02.01.2006") // Default to today
}