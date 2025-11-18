"""Profiles tab for managing launch profiles"""
from PySide6.QtWidgets import (
    QWidget, QVBoxLayout, QHBoxLayout, QListWidget,
    QPushButton, QLabel, QLineEdit, QSpinBox,
    QMessageBox, QGroupBox, QFormLayout, QComboBox
)
from PySide6.QtCore import Qt
from core.profile_manager import ProfileManager
from core.version_manager import VersionManager

class ProfilesTab(QWidget):
    """Tab for managing launch profiles"""
    
    def __init__(self, profile_manager: ProfileManager, version_manager: VersionManager):
        super().__init__()
        self.profile_manager = profile_manager
        self.version_manager = version_manager
        self.current_profile = None
        self.init_ui()
        self.load_profiles()
    
    def init_ui(self):
        """Initialize the UI"""
        layout = QHBoxLayout(self)
        
        # Left side - Profile list
        left_layout = QVBoxLayout()
        
        header = QLabel("Profiles")
        header.setStyleSheet("font-size: 16px; font-weight: bold;")
        left_layout.addWidget(header)
        
        self.profile_list = QListWidget()
        self.profile_list.currentItemChanged.connect(self.on_profile_selected)
        left_layout.addWidget(self.profile_list)
        
        buttons_layout = QHBoxLayout()
        self.new_button = QPushButton("New")
        self.new_button.clicked.connect(self.create_new_profile)
        buttons_layout.addWidget(self.new_button)
        
        self.delete_button = QPushButton("Delete")
        self.delete_button.clicked.connect(self.delete_profile)
        buttons_layout.addWidget(self.delete_button)
        left_layout.addLayout(buttons_layout)
        
        layout.addLayout(left_layout)
        
        # Right side - Profile details
        right_layout = QVBoxLayout()
        
        details_group = QGroupBox("Profile Details")
        form_layout = QFormLayout()
        
        self.name_edit = QLineEdit()
        form_layout.addRow("Name:", self.name_edit)
        
        self.version_combo = QComboBox()
        form_layout.addRow("Version:", self.version_combo)
        
        self.username_edit = QLineEdit()
        form_layout.addRow("Username:", self.username_edit)
        
        self.min_memory_spin = QSpinBox()
        self.min_memory_spin.setRange(512, 8192)
        self.min_memory_spin.setSuffix(" MB")
        form_layout.addRow("Min Memory:", self.min_memory_spin)
        
        self.max_memory_spin = QSpinBox()
        self.max_memory_spin.setRange(1024, 16384)
        self.max_memory_spin.setSuffix(" MB")
        form_layout.addRow("Max Memory:", self.max_memory_spin)
        
        details_group.setLayout(form_layout)
        right_layout.addWidget(details_group)
        
        buttons_layout = QHBoxLayout()
        self.save_button = QPushButton("Save Profile")
        self.save_button.clicked.connect(self.save_profile)
        buttons_layout.addWidget(self.save_button)
        
        self.set_default_button = QPushButton("Set as Default")
        self.set_default_button.clicked.connect(self.set_default_profile)
        buttons_layout.addWidget(self.set_default_button)
        
        right_layout.addLayout(buttons_layout)
        right_layout.addStretch()
        layout.addLayout(right_layout)
    
    def load_profiles(self):
        """Load profiles"""
        self.profile_list.clear()
        profiles = self.profile_manager.get_profiles()
        
        for name in profiles.keys():
            self.profile_list.addItem(name)
        
        # Load installed versions
        installed = self.version_manager.get_installed_versions()
        self.version_combo.clear()
        self.version_combo.addItems(installed if installed else ["No versions installed"])
    
    def on_profile_selected(self, item):
        """Handle profile selection"""
        if not item:
            return
        
        profile_name = item.text()
        profile = self.profile_manager.get_profile(profile_name)
        
        if profile:
            self.current_profile = profile_name
            self.name_edit.setText(profile.get("name", ""))
            self.username_edit.setText(profile.get("username", "Player"))
            self.min_memory_spin.setValue(profile.get("min_memory", 1024))
            self.max_memory_spin.setValue(profile.get("max_memory", 4096))
            
            # Set version in combo
            version = profile.get("version", "")
            index = self.version_combo.findText(version)
            if index >= 0:
                self.version_combo.setCurrentIndex(index)
    
    def create_new_profile(self):
        """Create a new profile"""
        name, ok = QMessageBox.getText(
            self,
            "New Profile",
            "Enter profile name:"
        )
        
        if ok and name:
            if name in self.profile_manager.get_profiles():
                QMessageBox.warning(self, "Error", "Profile with this name already exists.")
                return
            
            # Create with default values
            installed = self.version_manager.get_installed_versions()
            version = installed[0] if installed else "latest"
            
            self.profile_manager.create_profile(name, version)
            self.load_profiles()
            
            # Select the new profile
            items = self.profile_list.findItems(name, Qt.MatchExactly)
            if items:
                self.profile_list.setCurrentItem(items[0])
    
    def delete_profile(self):
        """Delete selected profile"""
        current_item = self.profile_list.currentItem()
        if not current_item:
            QMessageBox.warning(self, "No Selection", "Please select a profile to delete.")
            return
        
        profile_name = current_item.text()
        
        reply = QMessageBox.question(
            self,
            "Delete Profile",
            f"Are you sure you want to delete profile '{profile_name}'?",
            QMessageBox.Yes | QMessageBox.No
        )
        
        if reply == QMessageBox.Yes:
            self.profile_manager.delete_profile(profile_name)
            self.load_profiles()
            self.current_profile = None
    
    def save_profile(self):
        """Save profile changes"""
        if not self.current_profile:
            QMessageBox.warning(self, "No Profile", "Please select a profile first.")
            return
        
        name = self.name_edit.text()
        if not name:
            QMessageBox.warning(self, "Invalid Name", "Profile name cannot be empty.")
            return
        
        version = self.version_combo.currentText()
        username = self.username_edit.text() or "Player"
        min_memory = self.min_memory_spin.value()
        max_memory = self.max_memory_spin.value()
        
        # Update profile
        profile = self.profile_manager.get_profile(self.current_profile)
        if profile:
            profile["name"] = name
            profile["version"] = version
            profile["username"] = username
            profile["min_memory"] = min_memory
            profile["max_memory"] = max_memory
            
            # Save to config
            config = self.profile_manager.config.load_config()
            profiles = config.get("profiles", {})
            profiles[name] = profile
            config["profiles"] = profiles
            self.profile_manager.config.save_config(config)
            
            QMessageBox.information(self, "Saved", "Profile saved successfully.")
            self.load_profiles()
    
    def set_default_profile(self):
        """Set selected profile as default for launching"""
        current_item = self.profile_list.currentItem()
        if not current_item:
            QMessageBox.warning(self, "No Selection", "Please select a profile first.")
            return
        
        profile_name = current_item.text()
        config = self.profile_manager.config.load_config()
        config["selected_profile"] = profile_name
        self.profile_manager.config.save_config(config)
        
        QMessageBox.information(
            self,
            "Default Profile Set",
            f"'{profile_name}' is now the default launch profile."
        )

