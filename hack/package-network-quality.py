"""Build native release archives without third-party packaging dependencies."""
import os
from pathlib import Path
import shutil
import tarfile
import zipfile

root = Path(__file__).resolve().parent.parent
platform = os.environ['FRP_PLATFORM']
name = f'frp_0.71.1-network-quality.1_{platform}'
stage = root / 'release-stage' / name
stage.mkdir(parents=True, exist_ok=True)
ext = '.exe' if platform.startswith('windows') else ''
for binary in ('frps', 'frpc'):
    shutil.copy2(root / 'bin' / (binary + ext), stage)
for file in ('frps.toml', 'frpc.toml'):
    shutil.copy2(root / 'conf' / file, stage)
for file in ('LICENSE', 'Release.md'):
    shutil.copy2(root / file, stage)
shutil.copy2(root / 'doc' / 'network-quality.md', stage)
out = root / 'release'
out.mkdir(exist_ok=True)
if ext:
    with zipfile.ZipFile(out / (name + '.zip'), 'w', zipfile.ZIP_DEFLATED) as archive:
        for file in sorted(stage.iterdir()):
            archive.write(file, f'{name}/{file.name}')
else:
    with tarfile.open(out / (name + '.tar.gz'), 'w:gz') as archive:
        archive.add(stage, arcname=name)
