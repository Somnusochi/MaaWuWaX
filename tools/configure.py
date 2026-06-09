from pathlib import Path

import shutil
import urllib.request
import zipfile

assets_dir = Path(__file__).parent.parent.resolve() / "assets"
ocr_url = "https://download.maafw.xyz/MaaCommonAssets/OCR/ppocr_v5/ppocr_v5-zh_cn.zip"
ocr_required_files = ("det.onnx", "keys.txt", "rec.onnx")


def configure_ocr_model():
    ocr_dir = assets_dir / "resource" / "model" / "ocr"
    if all((ocr_dir / name).exists() for name in ocr_required_files):
        print("Found existing OCR directory, skipping default OCR model import.")
        return

    assets_ocr_dir = assets_dir / "MaaCommonAssets" / "OCR"
    common_ocr_dir = assets_ocr_dir / "ppocr_v5" / "zh_cn"
    if common_ocr_dir.exists():
        shutil.copytree(
            common_ocr_dir,
            ocr_dir,
            dirs_exist_ok=True,
        )
        return

    print(f"OCR model not found locally, downloading from {ocr_url}")
    ocr_dir.mkdir(parents=True, exist_ok=True)
    archive_path = ocr_dir.parent / "ppocr_v5-zh_cn.zip"
    urllib.request.urlretrieve(ocr_url, archive_path)
    with zipfile.ZipFile(archive_path) as archive:
        archive.extractall(ocr_dir)
    archive_path.unlink()

    missing_files = [name for name in ocr_required_files if not (ocr_dir / name).exists()]
    if missing_files:
        print(f"File Not Found: {ocr_dir} missing {', '.join(missing_files)}")
        exit(1)
    else:
        print("OCR model configured.")


if __name__ == "__main__":
    configure_ocr_model()

    print("OCR model configured.")
