#!/usr/bin/env python3
from pathlib import Path
import hashlib, json, base64, subprocess, zipfile

root = Path(__file__).resolve().parents[1]
build = root / "build"
dist = root / "dist"
dist.mkdir(exist_ok=True)
files = {
    "runtimes/linux-amd64/basispoints-transport": build / "linux-amd64/basispoints-transport",
    "runtimes/linux-arm64/basispoints-transport": build / "linux-arm64/basispoints-transport",
    "ui/index.html": root / "ui/index.html",
}
for p in files.values():
    if not p.exists():
        raise SystemExit(f"missing {p}")

src = json.loads((root / "manifest.source.json").read_text(encoding="utf-8"))
version = src["version"]
src["runtimes"] = {
    "linux-amd64": {"path": "runtimes/linux-amd64/basispoints-transport"},
    "linux-arm64": {"path": "runtimes/linux-arm64/basispoints-transport"},
}
src["files"] = {}
for name, p in files.items():
    src["files"][name] = hashlib.sha256(p.read_bytes()).hexdigest()
manifest = (json.dumps(src, ensure_ascii=False, separators=(",", ":")) + "\n").encode()
manifest_path = build / "manifest.json"
manifest_path.write_bytes(manifest)

key = root / ".publisher-key.pem"
sig = build / "signature.bin"
subprocess.run(["openssl", "pkeyutl", "-sign", "-rawin", "-inkey", str(key), "-in", str(manifest_path), "-out", str(sig)], check=True)
sigobj = {
    "algorithm": "ed25519",
    "key_id": "basispoints-local-v1",
    "signature": base64.b64encode(sig.read_bytes()).decode(),
}
sigbytes = (json.dumps(sigobj, separators=(",", ":")) + "\n").encode()

pubder = subprocess.check_output(["openssl", "pkey", "-in", str(key), "-pubout", "-outform", "DER"])
rawpub = pubder[-32:]
pub64 = base64.b64encode(rawpub).decode()
(dist / "trusted-publisher.yaml").write_text(
    "plugins:\n  allow_unsigned: false\n  trusted_publishers:\n    basispoints-local-v1: \"" + pub64 + "\"\n",
    encoding="utf-8",
)

out = dist / f"basispoints-transport-{version}.s2plugin"
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    z.writestr("manifest.json", manifest)
    z.writestr("signature.json", sigbytes)
    for name, p in files.items():
        zi = zipfile.ZipInfo(name)
        zi.external_attr = (0o755 if name.startswith("runtimes/") else 0o644) << 16
        zi.compress_type = zipfile.ZIP_DEFLATED
        z.writestr(zi, p.read_bytes())
print(out)
print(dist / "trusted-publisher.yaml")
