## 📋 API Endpoints

### GET /api/v1/ads
Returns a list of active ads with metadata.

**Response:**
```json
{
  "ads": [
    {
      "id": 1,
      "image_url": "https://example.com/ad1.jpg",
      "target_url": "https://example.com/product1",
      "title": "Amazing Product 1",
      "active": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### POST /api/v1/ads/click
Records a click event for an ad.

**Request:**
```json
{
  "ad_id": 1,
  "timestamp": 1704067200,
  "video_playback_time": 30
}
```

**Response:**
```json
{
  "status": "recorded"
}
```

### GET /api/v1/ads/analytics
Returns analytics data for ads.

**Query Parameters:**
- `ad_id` (optional): Specific ad ID
- `timeframe` (optional): `1h`, `24h`, `7d` (default: `24h`)

**Response:**
```json
{
  "analytics": [
    {
      "ad_id": 1,
      "click_count": 150,
      "last_hour": 12,
      "last_day": 150
    }
  ]
}
```
