"""Manages Minecraft profiles"""
import json
from pathlib import Path
from typing import List, Dict, Optional
from core.config import Config

class ProfileManager:
    """Handles Minecraft launch profiles"""
    
    def __init__(self, config: Config):
        self.config = config
        self.profiles_dir = config.profiles_dir
        
    def get_profiles(self) -> Dict[str, Dict]:
        """Get all profiles"""
        config = self.config.load_config()
        return config.get("profiles", {})
    
    def create_profile(self, name: str, version_id: str, 
                      java_path: Optional[str] = None,
                      min_memory: int = 1024,
                      max_memory: int = 4096) -> Dict:
        """Create a new profile"""
        config = self.config.load_config()
        
        profile = {
            "name": name,
            "version": version_id,
            "java_path": java_path or config.get("java_path", "java"),
            "min_memory": min_memory,
            "max_memory": max_memory,
            "game_directory": str(self.profiles_dir / name),
            "created": str(Path().cwd())  # Simple timestamp placeholder
        }
        
        profiles = config.get("profiles", {})
        profiles[name] = profile
        config["profiles"] = profiles
        
        self.config.save_config(config)
        
        # Create profile directory
        profile_dir = Path(profile["game_directory"])
        profile_dir.mkdir(parents=True, exist_ok=True)
        
        return profile
    
    def delete_profile(self, name: str) -> bool:
        """Delete a profile"""
        config = self.config.load_config()
        profiles = config.get("profiles", {})
        
        if name in profiles:
            del profiles[name]
            config["profiles"] = profiles
            self.config.save_config(config)
            return True
        return False
    
    def get_profile(self, name: str) -> Optional[Dict]:
        """Get a specific profile"""
        profiles = self.get_profiles()
        return profiles.get(name)

