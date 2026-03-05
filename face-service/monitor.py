"""Background monitor — polls cameras for faces, matches against gallery, reports sightings/visitors."""

import base64
import io
import logging
import os
import subprocess
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Thread

import face_recognition
import numpy as np
import requests
from PIL import Image

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger(__name__)

SURVEILLANCE_API_URL = os.environ.get("SURVEILLANCE_API_URL", "http://localhost:8080")
SURVEILLANCE_AUTH_PASSWORD = os.environ.get("SURVEILLANCE_AUTH_PASSWORD", "")
MATCH_THRESHOLD = float(os.environ.get("FACE_MATCH_THRESHOLD", "0.6"))
SNAPSHOT_WORKERS = int(os.environ.get("SNAPSHOT_WORKERS", "4"))


def api_session():
    s = requests.Session()
    if SURVEILLANCE_AUTH_PASSWORD:
        s.auth = ("", SURVEILLANCE_AUTH_PASSWORD)
    return s


class MonitorLoop:
    def __init__(self):
        self.session = api_session()
        self.gallery = []
        self.gallery_ts = 0
        self.configs = []
        self.configs_ts = 0
        self.running = False

    def refresh_gallery(self):
        if time.time() - self.gallery_ts < 30:
            return
        try:
            resp = self.session.get(f"{SURVEILLANCE_API_URL}/api/faces/internal/gallery")
            resp.raise_for_status()
            self.gallery = resp.json()
            self.gallery_ts = time.time()
            log.info("Refreshed gallery: %d embeddings", len(self.gallery))
        except Exception as e:
            log.error("Failed to refresh gallery: %s", e)

    def refresh_configs(self):
        if time.time() - self.configs_ts < 30:
            return
        try:
            resp = self.session.get(f"{SURVEILLANCE_API_URL}/api/faces/config")
            resp.raise_for_status()
            self.configs = resp.json()
            self.configs_ts = time.time()
            log.info("Refreshed configs: %d cameras monitored", len(self.configs))
        except Exception as e:
            log.error("Failed to refresh configs: %s", e)

    def fetch_rtsp_snapshot(self, rtsp_url):
        """Grab a single JPEG frame directly from RTSP via ffmpeg."""
        try:
            result = subprocess.run(
                [
                    "ffmpeg", "-rtsp_transport", "tcp",
                    "-i", rtsp_url,
                    "-frames:v", "1",
                    "-f", "image2", "-q:v", "3",
                    "-update", "1",
                    "-y", "pipe:1",
                ],
                capture_output=True,
                timeout=10,
            )
            if result.returncode != 0:
                log.error("ffmpeg snapshot failed: %s", result.stderr[-200:].decode(errors="replace"))
                return None
            if len(result.stdout) < 1000:
                log.warning("ffmpeg snapshot too small (%d bytes)", len(result.stdout))
                return None
            return result.stdout
        except subprocess.TimeoutExpired:
            log.error("ffmpeg snapshot timed out")
            return None
        except Exception as e:
            log.error("ffmpeg snapshot error: %s", e)
            return None

    def fetch_snapshot_for_config(self, cfg):
        """Fetch snapshot for a camera config. Returns (cfg, jpeg_data)."""
        rtsp_url = cfg.get("rtsp_url")
        if rtsp_url:
            return cfg, self.fetch_rtsp_snapshot(rtsp_url)
        # Fallback to Go proxy
        camera_id = cfg["camera_id"]
        try:
            resp = self.session.get(
                f"{SURVEILLANCE_API_URL}/api/faces/snapshots/{camera_id}", timeout=15
            )
            resp.raise_for_status()
            return cfg, resp.content
        except Exception as e:
            log.error("Snapshot failed for %s: %s", camera_id, e)
            return cfg, None

    def detect_faces(self, jpeg_data):
        try:
            image = face_recognition.load_image_file(io.BytesIO(jpeg_data))
        except Exception as e:
            log.error("Failed to load image: %s", e)
            return []
        locations = face_recognition.face_locations(image, model="hog")
        if not locations:
            return []

        encodings = face_recognition.face_encodings(image, locations)
        results = []
        h, w = image.shape[:2]

        for loc, enc in zip(locations, encodings):
            top, right, bottom, left = loc
            pad = int((bottom - top) * 0.3)
            crop = image[
                max(0, top - pad) : min(h, bottom + pad),
                max(0, left - pad) : min(w, right + pad),
            ]

            pil_crop = Image.fromarray(crop)
            pil_crop.thumbnail((200, 200))
            buf = io.BytesIO()
            pil_crop.save(buf, format="JPEG", quality=80)
            crop_b64 = base64.b64encode(buf.getvalue()).decode()

            results.append({"embedding": enc, "crop_base64": crop_b64})

        return results

    def match_gallery(self, embedding):
        """Match embedding against gallery. Returns (subject_id, confidence) or None."""
        if not self.gallery:
            return None

        gallery_embeddings = np.array([g["embedding"] for g in self.gallery])
        distances = face_recognition.face_distance(gallery_embeddings, embedding)

        best_idx = np.argmin(distances)
        best_dist = distances[best_idx]

        if best_dist < MATCH_THRESHOLD:
            entry = self.gallery[best_idx]
            if entry.get("alert_enabled", True):
                confidence = 1.0 - best_dist
                return entry["subject_id"], confidence
        return None

    def report_sighting(self, subject_id, camera_id, confidence, crop_b64):
        try:
            self.session.post(
                f"{SURVEILLANCE_API_URL}/api/faces/internal/sighting",
                json={
                    "subject_id": subject_id,
                    "camera_id": camera_id,
                    "confidence": confidence,
                    "crop_base64": crop_b64,
                },
            )
        except Exception as e:
            log.error("Failed to report sighting: %s", e)

    def report_visitor(self, camera_id, embedding, crop_b64):
        try:
            self.session.post(
                f"{SURVEILLANCE_API_URL}/api/faces/internal/visitor",
                json={
                    "camera_id": camera_id,
                    "embedding": embedding.tolist(),
                    "crop_base64": crop_b64,
                },
            )
        except Exception as e:
            log.error("Failed to report visitor: %s", e)

    def process_faces(self, camera_id, monitor_type, jpeg_data):
        """Detect faces and handle matches/visitors."""
        faces = self.detect_faces(jpeg_data)
        log.info("Camera %s: detected %d face(s)", camera_id, len(faces))
        if not faces:
            return

        for face in faces:
            if monitor_type in ("realtime", "both"):
                match = self.match_gallery(face["embedding"])
                if match:
                    subject_id, confidence = match
                    log.info(
                        "MATCH: subject=%s camera=%s confidence=%.2f",
                        subject_id,
                        camera_id,
                        confidence,
                    )
                    self.report_sighting(
                        subject_id, camera_id, confidence, face["crop_base64"]
                    )
            if monitor_type in ("batch", "both"):
                self.report_visitor(camera_id, face["embedding"], face["crop_base64"])

    def run(self):
        self.running = True
        log.info("Monitor loop started (workers=%d)", SNAPSHOT_WORKERS)

        while self.running:
            self.refresh_gallery()
            self.refresh_configs()

            if not self.configs:
                time.sleep(5)
                continue

            cycle_start = time.time()

            # Phase 1: Fetch all snapshots in parallel
            snapshots = []
            with ThreadPoolExecutor(max_workers=SNAPSHOT_WORKERS) as pool:
                futures = {
                    pool.submit(self.fetch_snapshot_for_config, cfg): cfg
                    for cfg in self.configs
                }
                for future in as_completed(futures):
                    if not self.running:
                        break
                    try:
                        cfg, jpeg_data = future.result()
                        if jpeg_data:
                            snapshots.append((cfg, jpeg_data))
                    except Exception as e:
                        log.error("Snapshot future error: %s", e)

            fetch_elapsed = time.time() - cycle_start

            # Phase 2: Process faces sequentially (CPU-bound)
            for cfg, jpeg_data in snapshots:
                if not self.running:
                    break
                try:
                    self.process_faces(cfg["camera_id"], cfg["monitor_type"], jpeg_data)
                except Exception as e:
                    log.error("Error processing camera %s: %s", cfg["camera_id"], e)

            elapsed = time.time() - cycle_start
            log.info(
                "Cycle complete: %d cameras, fetch=%.1fs, total=%.1fs",
                len(self.configs), fetch_elapsed, elapsed,
            )

            min_interval = self.configs[0].get("interval_seconds", 2) if self.configs else 2
            remaining = min_interval - elapsed
            if remaining > 0:
                time.sleep(remaining)

    def stop(self):
        self.running = False


def start_monitor_thread():
    monitor = MonitorLoop()
    thread = Thread(target=monitor.run, daemon=True)
    thread.start()
    return monitor


if __name__ == "__main__":
    monitor = MonitorLoop()
    try:
        monitor.run()
    except KeyboardInterrupt:
        monitor.stop()
