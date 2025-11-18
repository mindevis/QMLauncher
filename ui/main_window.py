"""Main window for QMLauncher"""
from PySide6.QtWidgets import (
    QMainWindow, QWidget, QVBoxLayout, QHBoxLayout,
    QTabWidget, QPushButton, QLabel, QListWidget,
    QListWidgetItem, QLineEdit, QSpinBox, QFileDialog,
    QMessageBox, QGroupBox, QFormLayout
)
from PySide6.QtCore import Qt, QThread, Signal
from PySide6.QtGui import QFont

from core.config import Config
from core.version_manager import VersionManager
from core.profile_manager import ProfileManager
from core.game_launcher import GameLauncher
from ui.versions_tab import VersionsTab
from ui.profiles_tab import ProfilesTab
from ui.settings_tab import SettingsTab

class MainWindow(QMainWindow):
    """Main application window"""
    
    def __init__(self):
        super().__init__()
        self.config = Config()
        self.config.ensure_directories()
        
        self.version_manager = VersionManager(self.config)
        self.profile_manager = ProfileManager(self.config)
        self.game_launcher = GameLauncher(self.config, self.version_manager)
        
        self.init_ui()
        self.load_config()
    
    def init_ui(self):
        """Initialize the UI"""
        self.setWindowTitle("QMLauncher - Minecraft Launcher")
        self.setMinimumSize(900, 600)
        
        # Central widget with tabs
        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        
        layout = QVBoxLayout(central_widget)
        
        # Header
        header = QLabel("QMLauncher")
        header_font = QFont()
        header_font.setPointSize(24)
        header_font.setBold(True)
        header.setFont(header_font)
        header.setAlignment(Qt.AlignCenter)
        layout.addWidget(header)
        
        # Tabs
        self.tabs = QTabWidget()
        self.tabs.addTab(VersionsTab(self.version_manager), "Versions")
        self.tabs.addTab(ProfilesTab(self.profile_manager, self.version_manager), "Profiles")
        self.tabs.addTab(SettingsTab(self.config), "Settings")
        layout.addWidget(self.tabs)
        
        # Launch button
        launch_layout = QHBoxLayout()
        launch_layout.addStretch()
        
        self.launch_button = QPushButton("Launch Minecraft")
        self.launch_button.setMinimumHeight(40)
        self.launch_button.clicked.connect(self.launch_game)
        launch_layout.addWidget(self.launch_button)
        
        launch_layout.addStretch()
        layout.addLayout(launch_layout)
    
    def load_config(self):
        """Load configuration"""
        config = self.config.load_config()
        # Apply any window settings from config
        width = config.get("window_width", 900)
        height = config.get("window_height", 600)
        self.resize(width, height)
    
    def launch_game(self):
        """Launch Minecraft"""
        config = self.config.load_config()
        selected_profile = config.get("selected_profile")
        
        if not selected_profile:
            QMessageBox.warning(
                self,
                "No Profile Selected",
                "Please select a profile in the Profiles tab."
            )
            return
        
        profile = self.profile_manager.get_profile(selected_profile)
        if not profile:
            QMessageBox.warning(
                self,
                "Profile Not Found",
                f"Profile '{selected_profile}' not found."
            )
            return
        
        # Launch game
        if self.game_launcher.launch(selected_profile):
            QMessageBox.information(
                self,
                "Launching",
                f"Launching Minecraft with profile '{selected_profile}'..."
            )
        else:
            QMessageBox.critical(
                self,
                "Launch Failed",
                "Failed to launch Minecraft. Check the console for details."
            )
    
    def closeEvent(self, event):
        """Save window size on close"""
        config = self.config.load_config()
        config["window_width"] = self.width()
        config["window_height"] = self.height()
        self.config.save_config(config)
        event.accept()

