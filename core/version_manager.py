"""Manages Minecraft versions"""
import json
import requests
from pathlib import Path
from typing import List, Dict, Optional
from core.config import Config

class VersionManager:
    """Handles downloading and managing Minecraft versions"""
    
    VERSION_MANIFEST_URL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
    
    def __init__(self, config: Config):
        self.config = config
        self.versions_dir = config.versions_dir
        
    def get_version_manifest(self) -> Optional[Dict]:
        """Fetch version manifest from Mojang"""
        try:
            response = requests.get(self.VERSION_MANIFEST_URL, timeout=10)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"Error fetching version manifest: {e}")
            return None
    
    def get_available_versions(self) -> List[Dict]:
        """Get list of available Minecraft versions"""
        manifest = self.get_version_manifest()
        if not manifest:
            return []
        
        versions = []
        for version in manifest.get("versions", []):
            versions.append({
                "id": version["id"],
                "type": version["type"],
                "releaseTime": version["releaseTime"]
            })
        return versions
    
    def get_installed_versions(self) -> List[str]:
        """Get list of installed versions"""
        if not self.versions_dir.exists():
            return []
        
        installed = []
        for version_dir in self.versions_dir.iterdir():
            if version_dir.is_dir():
                json_file = version_dir / f"{version_dir.name}.json"
                if json_file.exists():
                    installed.append(version_dir.name)
        return sorted(installed, reverse=True)
    
    def is_version_installed(self, version_id: str) -> bool:
        """Check if a version is installed"""
        version_dir = self.versions_dir / version_id
        json_file = version_dir / f"{version_id}.json"
        return json_file.exists()
    
    def get_version_info(self, version_id: str) -> Optional[Dict]:
        """Get version information"""
        version_dir = self.versions_dir / version_id
        json_file = version_dir / f"{version_id}.json"
        
        if json_file.exists():
            with open(json_file, 'r', encoding='utf-8') as f:
                return json.load(f)
        return None

