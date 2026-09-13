ALTER TABLE site_settings ADD COLUMN youtube_video_url VARCHAR(500) NOT NULL DEFAULT '';
UPDATE site_settings SET youtube_video_url='https://www.youtube.com/watch?v=jNQXAC9IVRw' WHERE singleton=TRUE;