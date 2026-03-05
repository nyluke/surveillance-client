import { useState, useRef } from "react";
import { useFaceStore } from "../../stores/faceStore";

export default function AddSubjectDialog({
  onClose,
}: {
  onClose: () => void;
}) {
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [preview, setPreview] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const { createSubject, loading, error } = useFaceStore();

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setPreview(URL.createObjectURL(file));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const file = fileRef.current?.files?.[0];
    if (!file || !name) return;

    const formData = new FormData();
    formData.append("name", name);
    formData.append("notes", notes);
    formData.append("image", file);

    await createSubject(formData);
    if (!useFaceStore.getState().error) {
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">Add Person to Watchlist</h3>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-gray-700 rounded px-3 py-2 text-sm"
              required
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">
              Notes (optional)
            </label>
            <input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full bg-gray-700 rounded px-3 py-2 text-sm"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">
              Photo (clear face)
            </label>
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              onChange={handleFile}
              className="w-full text-sm"
              required
            />
          </div>

          {preview && (
            <div className="flex justify-center">
              <img
                src={preview}
                alt="Preview"
                className="max-h-48 rounded"
              />
            </div>
          )}

          {error && <p className="text-red-400 text-sm">{error}</p>}

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-400 hover:text-white"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !name}
              className="px-4 py-2 bg-blue-600 rounded text-sm font-medium disabled:opacity-50"
            >
              {loading ? "Enrolling..." : "Add Person"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
