"""Versions tab for managing Minecraft versions"""
from PySide6.QtWidgets import (
    QWidget, QVBoxLayout, QHBoxLayout, QListWidget, QListWidgetItem,
    QPushButton, QLabel, QMessageBox
)
from PySide6.QtCore import QThread, Signal, Qt
from core.version_manager import VersionManager

class VersionFetchThread(QThread):
    """Thread for fetching versions"""
    versions_fetched = Signal(list)
    
    def __init__(self, version_manager: VersionManager):
        super().__init__()
        self.version_manager = version_manager
    
    def run(self):
        versions = self.version_manager.get_available_versions()
        self.versions_fetched.emit(versions)

class VersionsTab(QWidget):
    """Tab for managing Minecraft versions"""
    
    def __init__(self, version_manager: VersionManager):
        super().__init__()
        self.version_manager = version_manager
        self.init_ui()
        self.load_versions()
    
    def init_ui(self):
        """Initialize the UI"""
        layout = QVBoxLayout(self)
        
        # Header
        header = QLabel("Minecraft Versions")
        header.setStyleSheet("font-size: 16px; font-weight: bold;")
        layout.addWidget(header)
        
        # Versions list
        list_layout = QHBoxLayout()
        
        self.available_list = QListWidget()
        self.available_list.setMinimumHeight(300)
        list_layout.addWidget(self.available_list)
        
        self.installed_list = QListWidget()
        self.installed_list.setMinimumHeight(300)
        list_layout.addWidget(self.installed_list)
        
        layout.addLayout(list_layout)
        
        # Labels
        labels_layout = QHBoxLayout()
        labels_layout.addWidget(QLabel("Available Versions"))
        labels_layout.addWidget(QLabel("Installed Versions"))
        layout.addLayout(labels_layout)
        
        # Buttons
        buttons_layout = QHBoxLayout()
        
        self.refresh_button = QPushButton("Refresh")
        self.refresh_button.clicked.connect(self.load_versions)
        buttons_layout.addWidget(self.refresh_button)
        
        self.install_button = QPushButton("Install Selected")
        self.install_button.clicked.connect(self.install_version)
        buttons_layout.addWidget(self.install_button)
        
        buttons_layout.addStretch()
        layout.addLayout(buttons_layout)
    
    def load_versions(self):
        """Load available and installed versions"""
        # Load installed versions
        installed = self.version_manager.get_installed_versions()
        self.installed_list.clear()
        for version in installed:
            self.installed_list.addItem(version)
        
        # Fetch available versions in thread
        self.refresh_button.setEnabled(False)
        self.refresh_button.setText("Loading...")
        
        self.fetch_thread = VersionFetchThread(self.version_manager)
        self.fetch_thread.versions_fetched.connect(self.on_versions_fetched)
        self.fetch_thread.start()
    
    def on_versions_fetched(self, versions):
        """Handle versions fetched"""
        self.available_list.clear()
        
        # Filter to show only release and snapshot versions
        for version in versions:
            if version["type"] in ["release", "snapshot"]:
                item_text = f"{version['id']} ({version['type']})"
                item = QListWidgetItem(item_text)
                item.setData(Qt.UserRole, version["id"])
                self.available_list.addItem(item)
        
        self.refresh_button.setEnabled(True)
        self.refresh_button.setText("Refresh")
    
    def install_version(self):
        """Install selected version"""
        current_item = self.available_list.currentItem()
        if not current_item:
            QMessageBox.warning(self, "No Selection", "Please select a version to install.")
            return
        
        version_id = current_item.data(Qt.UserRole)
        if not version_id:
            version_id = current_item.text().split()[0]
        
        if self.version_manager.is_version_installed(version_id):
            QMessageBox.information(
                self,
                "Already Installed",
                f"Version '{version_id}' is already installed."
            )
            return
        
        # TODO: Implement actual version installation
        QMessageBox.information(
            self,
            "Installation",
            f"Version installation for '{version_id}' will be implemented soon."
        )

