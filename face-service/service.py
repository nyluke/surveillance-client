"""Face recognition service — Flask app with enrollment, detection, and clustering."""

import base64
import io
import os

import face_recognition
import numpy as np
from flask import Flask, jsonify, request
from PIL import Image
from sklearn.cluster import DBSCAN

app = Flask(__name__)


@app.route("/health")
def health():
    return jsonify({"status": "ok"})


@app.route("/enroll", methods=["POST"])
def enroll():
    """Detect the largest face in an uploaded image, return embedding + crop."""
    if "image" not in request.files:
        return jsonify({"error": "image file required"}), 400

    file = request.files["image"]
    image = face_recognition.load_image_file(file)
    locations = face_recognition.face_locations(image, model="hog")

    if not locations:
        return jsonify({"error": "no face detected"}), 422

    # Pick largest face by area
    largest = max(locations, key=lambda loc: (loc[2] - loc[0]) * (loc[1] - loc[3]))
    encodings = face_recognition.face_encodings(image, [largest])

    if not encodings:
        return jsonify({"error": "failed to encode face"}), 422

    # Crop face with padding
    top, right, bottom, left = largest
    h, w = image.shape[:2]
    pad = int((bottom - top) * 0.3)
    crop = image[
        max(0, top - pad) : min(h, bottom + pad),
        max(0, left - pad) : min(w, right + pad),
    ]

    # Encode crop as JPEG base64
    pil_crop = Image.fromarray(crop)
    pil_crop.thumbnail((300, 300))
    buf = io.BytesIO()
    pil_crop.save(buf, format="JPEG", quality=85)
    crop_b64 = base64.b64encode(buf.getvalue()).decode()

    return jsonify(
        {
            "embedding": encodings[0].tolist(),
            "crop_base64": crop_b64,
        }
    )


@app.route("/detect", methods=["POST"])
def detect():
    """Detect all faces in an uploaded image, return embeddings + crops + bboxes."""
    if "image" not in request.files:
        return jsonify({"error": "image file required"}), 400

    file = request.files["image"]
    image = face_recognition.load_image_file(file)
    locations = face_recognition.face_locations(image, model="hog")

    if not locations:
        return jsonify([])

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

        results.append(
            {
                "embedding": enc.tolist(),
                "crop_base64": crop_b64,
                "bbox": {"top": top, "right": right, "bottom": bottom, "left": left},
            }
        )

    return jsonify(results)


@app.route("/cluster", methods=["POST"])
def cluster():
    """Run DBSCAN clustering on a set of embeddings."""
    data = request.get_json()
    if not data or "embeddings" not in data:
        return jsonify({"error": "embeddings required"}), 400

    embeddings = np.array(data["embeddings"])
    if len(embeddings) < 2:
        return jsonify({"labels": [0] * len(embeddings)})

    clustering = DBSCAN(eps=0.5, min_samples=2, metric="euclidean").fit(embeddings)
    return jsonify({"labels": clustering.labels_.tolist()})


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 5050))
    app.run(host="0.0.0.0", port=port, debug=False)
