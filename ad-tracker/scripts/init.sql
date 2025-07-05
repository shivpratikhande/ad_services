-- Initialize database with extensions and initial data
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_ads_active ON ads(active);
CREATE INDEX IF NOT EXISTS idx_ads_created_at ON ads(created_at);

CREATE INDEX IF NOT EXISTS idx_click_events_ad_id ON click_events(ad_id);
CREATE INDEX IF NOT EXISTS idx_click_events_timestamp ON click_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_click_events_processed ON click_events(processed);
CREATE INDEX IF NOT EXISTS idx_click_events_ip_address ON click_events(ip_address);

-- Create composite indexes for analytics queries
CREATE INDEX IF NOT EXISTS idx_click_events_ad_timestamp ON click_events(ad_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_click_events_timestamp_processed ON click_events(timestamp, processed);

-- Insert sample ads data
INSERT INTO ads (image_url, target_url, title, active, created_at, updated_at) 
VALUES 
  ('https://picsum.photos/300/200?random=1', 'https://example.com/product1', 'Amazing Product 1', true, NOW(), NOW()),
  ('https://picsum.photos/300/200?random=2', 'https://example.com/service2', 'Great Service 2', true, NOW(), NOW()),
  ('https://picsum.photos/300/200?random=3', 'https://example.com/offer3', 'Special Offer 3', true, NOW(), NOW()),
  ('https://picsum.photos/300/200?random=4', 'https://example.com/deal4', 'Best Deal 4', true, NOW(), NOW()),
  ('https://picsum.photos/300/200?random=5', 'https://example.com/promo5', 'Limited Promo 5', false, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Insert sample click events for testing
INSERT INTO click_events (ad_id, timestamp, ip_address, video_playback_time, user_agent, processed, created_at)
VALUES 
  (1, NOW() - INTERVAL '1 hour', '192.168.1.100', 30, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36', true, NOW()),
  (1, NOW() - INTERVAL '30 minutes', '192.168.1.101', 45, 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36', true, NOW()),
  (2, NOW() - INTERVAL '45 minutes', '192.168.1.102', 60, 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15', true, NOW()),
  (3, NOW() - INTERVAL '20 minutes', '192.168.1.103', 120, 'Mozilla/5.0 (Android 11; Mobile; rv:68.0) Gecko/68.0 Firefox/88.0', true, NOW()),
  (1, NOW() - INTERVAL '10 minutes', '192.168.1.104', 90, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36', true, NOW())
ON CONFLICT DO NOTHING;