export default function StatusBadge({ enabled }: { enabled: boolean }) {
  return (
    <span
      className={`inline-block w-2 h-2 rounded-full ${
        enabled ? "bg-green-500" : "bg-red-500"
      }`}
      title={enabled ? "Enabled" : "Disabled"}
    />
  );
}
