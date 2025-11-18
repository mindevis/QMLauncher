"""Configuration management for QMLauncher"""
from pathlib import Path
from typing import Optional
import json

class Config:
    """Manages launcher configuration and directories"""
    
    def __init__(self):
        self.base_dir = Path.home() / ".qmlauncher"
        self.versions_dir = self.base_dir / "versions"
        self.profiles_dir = self.base_dir / "profiles"
        self.assets_dir = self.base_dir / "assets"
        self.libraries_dir = self.base_dir / "libraries"
        self.config_file = self.base_dir / "config.json"
        
    def ensure_directories(self):
        """Create all necessary directories if they don't exist"""
        for directory in [self.base_dir, self.versions_dir, self.profiles_dir, 
                         self.assets_dir, self.libraries_dir]:
            directory.mkdir(parents=True, exist_ok=True)
    
    def load_config(self) -> dict:
        """Load configuration from file"""
        if self.config_file.exists():
            with open(self.config_file, 'r', encoding='utf-8') as f:
                return json.load(f)
        return self.get_default_config()
    
    def save_config(self, config: dict):
        """Save configuration to file"""
        with open(self.config_file, 'w', encoding='utf-8') as f:
            json.dump(config, f, indent=2, ensure_ascii=False)
    
    def get_default_config(self) -> dict:
        """Get default configuration"""
        return {
            "java_path": "java",
            "min_memory": 1024,
            "max_memory": 4096,
            "window_width": 1024,
            "window_height": 768,
            "selected_profile": None,
            "profiles": {}
        }

