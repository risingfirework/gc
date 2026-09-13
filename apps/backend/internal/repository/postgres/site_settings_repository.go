package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"tka/apps/backend/internal/domain"
)

const siteSettingsColumns = `platform_name,platform_tagline,logo_data_url,favicon_data_url,support_email,whatsapp,instagram_url,youtube_url,youtube_videos,hero_interval_ms,catalog_interval_ms,default_package_validity_days,default_exam_duration_minutes,default_exam_total_questions,default_passing_score,hero_slides,updated_at`

func (r *AdminRepository) GetSiteSettings(ctx context.Context) (*domain.SiteSettings, error) {
	var item domain.SiteSettings
	var slides, videos []byte
	if err := r.db.QueryRow(ctx, `SELECT `+siteSettingsColumns+` FROM site_settings WHERE singleton=TRUE`).Scan(&item.PlatformName, &item.PlatformTagline, &item.LogoDataURL, &item.FaviconDataURL, &item.SupportEmail, &item.WhatsApp, &item.InstagramURL, &item.YouTubeURL, &videos, &item.HeroIntervalMS, &item.CatalogIntervalMS, &item.DefaultPackageValidityDays, &item.DefaultExamDurationMinutes, &item.DefaultExamTotalQuestions, &item.DefaultPassingScore, &slides, &item.UpdatedAt); err != nil {
		return nil, fmt.Errorf("site settings: %w", err)
	}
	if err := json.Unmarshal(slides, &item.HeroSlides); err != nil {
		return nil, fmt.Errorf("decode hero slides: %w", err)
	}
	if err := json.Unmarshal(videos, &item.YouTubeVideos); err != nil {
		return nil, fmt.Errorf("decode youtube videos: %w", err)
	}
	return &item, nil
}

func (r *AdminRepository) UpdateSiteSettings(ctx context.Context, input domain.SiteSettings) (*domain.SiteSettings, error) {
	slides, err := json.Marshal(input.HeroSlides)
	if err != nil {
		return nil, fmt.Errorf("encode hero slides: %w", err)
	}
	videos, err := json.Marshal(input.YouTubeVideos)
	if err != nil {
		return nil, fmt.Errorf("encode youtube videos: %w", err)
	}
	const query = `UPDATE site_settings SET platform_name=$1,platform_tagline=$2,logo_data_url=$3,favicon_data_url=$4,support_email=$5,whatsapp=$6,instagram_url=$7,youtube_url=$8,youtube_videos=$9,hero_interval_ms=$10,catalog_interval_ms=$11,default_package_validity_days=$12,default_exam_duration_minutes=$13,default_exam_total_questions=$14,default_passing_score=$15,hero_slides=$16,updated_at=NOW() WHERE singleton=TRUE RETURNING ` + siteSettingsColumns
	var item domain.SiteSettings
	var rawSlides, rawVideos []byte
	if err := r.db.QueryRow(ctx, query, input.PlatformName, input.PlatformTagline, input.LogoDataURL, input.FaviconDataURL, input.SupportEmail, input.WhatsApp, input.InstagramURL, input.YouTubeURL, videos, input.HeroIntervalMS, input.CatalogIntervalMS, input.DefaultPackageValidityDays, input.DefaultExamDurationMinutes, input.DefaultExamTotalQuestions, input.DefaultPassingScore, slides).Scan(&item.PlatformName, &item.PlatformTagline, &item.LogoDataURL, &item.FaviconDataURL, &item.SupportEmail, &item.WhatsApp, &item.InstagramURL, &item.YouTubeURL, &rawVideos, &item.HeroIntervalMS, &item.CatalogIntervalMS, &item.DefaultPackageValidityDays, &item.DefaultExamDurationMinutes, &item.DefaultExamTotalQuestions, &item.DefaultPassingScore, &rawSlides, &item.UpdatedAt); err != nil {
		return nil, fmt.Errorf("update site settings: %w", err)
	}
	if err := json.Unmarshal(rawSlides, &item.HeroSlides); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawVideos, &item.YouTubeVideos); err != nil {
		return nil, err
	}
	return &item, nil
}
