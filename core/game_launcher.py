"""Handles launching Minecraft"""
import subprocess
from pathlib import Path
from typing import Dict, List
from core.config import Config
from core.version_manager import VersionManager

class GameLauncher:
    """Launches Minecraft with specified profile"""
    
    def __init__(self, config: Config, version_manager: VersionManager):
        self.config = config
        self.version_manager = version_manager
        
    def build_java_args(self, profile: Dict) -> List[str]:
        """Build Java command line arguments"""
        java_path = profile.get("java_path", "java")
        min_memory = profile.get("min_memory", 1024)
        max_memory = profile.get("max_memory", 4096)
        
        args = [
            java_path,
            f"-Xms{min_memory}M",
            f"-Xmx{max_memory}M",
            "-Djava.library.path=natives",
            "-cp", "libraries/*:minecraft.jar",
            "net.minecraft.client.main.Main",
            "--username", profile.get("username", "Player"),
            "--version", profile.get("version", "latest"),
            "--gameDir", profile.get("game_directory", "."),
            "--assetsDir", str(self.config.assets_dir),
            "--assetIndex", "1.20",
            "--uuid", "00000000-0000-0000-0000-000000000000",
            "--accessToken", "null",
            "--userType", "legacy",
            "--versionType", "release"
        ]
        
        return args
    
    def launch(self, profile_name: str) -> bool:
        """Launch Minecraft with the specified profile"""
        from core.profile_manager import ProfileManager
        profile_manager = ProfileManager(self.config)
        
        profile = profile_manager.get_profile(profile_name)
        if not profile:
            print(f"Profile '{profile_name}' not found")
            return False
        
        version_id = profile.get("version")
        if not self.version_manager.is_version_installed(version_id):
            print(f"Version '{version_id}' is not installed")
            return False
        
        # Build command
        args = self.build_java_args(profile)
        
        # Change to game directory
        game_dir = Path(profile.get("game_directory", "."))
        game_dir.mkdir(parents=True, exist_ok=True)
        
        try:
            # Launch Minecraft
            subprocess.Popen(
                args,
                cwd=str(game_dir),
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE
            )
            return True
        except Exception as e:
            print(f"Error launching Minecraft: {e}")
            return False

