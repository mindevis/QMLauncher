"""Settings tab for launcher configuration"""
from PySide6.QtWidgets import (
    QWidget, QVBoxLayout, QHBoxLayout, QLabel,
    QLineEdit, QPushButton, QFileDialog, QMessageBox,
    QGroupBox, QFormLayout, QSpinBox
)
from core.config import Config

class SettingsTab(QWidget):
    """Tab for launcher settings"""
    
    def __init__(self, config: Config):
        super().__init__()
        self.config = config
        self.init_ui()
        self.load_settings()
    
    def init_ui(self):
        """Initialize the UI"""
        layout = QVBoxLayout(self)
        
        # Java Settings
        java_group = QGroupBox("Java Settings")
        form_layout = QFormLayout()
        
        java_layout = QHBoxLayout()
        self.java_path_edit = QLineEdit()
        java_layout.addWidget(self.java_path_edit)
        
        self.browse_java_button = QPushButton("Browse...")
        self.browse_java_button.clicked.connect(self.browse_java)
        java_layout.addWidget(self.browse_java_button)
        
        form_layout.addRow("Java Path:", java_layout)
        java_group.setLayout(form_layout)
        layout.addWidget(java_group)
        
        # Memory Settings
        memory_group = QGroupBox("Default Memory Settings")
        memory_form = QFormLayout()
        
        self.default_min_memory = QSpinBox()
        self.default_min_memory.setRange(512, 8192)
        self.default_min_memory.setSuffix(" MB")
        memory_form.addRow("Default Min Memory:", self.default_min_memory)
        
        self.default_max_memory = QSpinBox()
        self.default_max_memory.setRange(1024, 16384)
        self.default_max_memory.setSuffix(" MB")
        memory_form.addRow("Default Max Memory:", self.default_max_memory)
        
        memory_group.setLayout(memory_form)
        layout.addWidget(memory_group)
        
        # Save button
        save_layout = QHBoxLayout()
        save_layout.addStretch()
        
        self.save_button = QPushButton("Save Settings")
        self.save_button.clicked.connect(self.save_settings)
        save_layout.addWidget(self.save_button)
        
        save_layout.addStretch()
        layout.addLayout(save_layout)
        
        layout.addStretch()
    
    def load_settings(self):
        """Load settings from config"""
        config = self.config.load_config()
        
        self.java_path_edit.setText(config.get("java_path", "java"))
        self.default_min_memory.setValue(config.get("min_memory", 1024))
        self.default_max_memory.setValue(config.get("max_memory", 4096))
    
    def browse_java(self):
        """Browse for Java executable"""
        file_path, _ = QFileDialog.getOpenFileName(
            self,
            "Select Java Executable",
            "",
            "Executable (*.exe);;All Files (*)"
        )
        
        if file_path:
            self.java_path_edit.setText(file_path)
    
    def save_settings(self):
        """Save settings to config"""
        config = self.config.load_config()
        
        config["java_path"] = self.java_path_edit.text()
        config["min_memory"] = self.default_min_memory.value()
        config["max_memory"] = self.default_max_memory.value()
        
        self.config.save_config(config)
        QMessageBox.information(self, "Saved", "Settings saved successfully.")

