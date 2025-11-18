import sys
import os
from pathlib import Path
from PySide6.QtWidgets import QApplication
from PySide6.QtCore import QDir

# Добавляем путь к модулям
sys.path.insert(0, str(Path(__file__).parent))

from ui.main_window import MainWindow
from core.config import Config

def main():
    # Создаем необходимые директории
    config = Config()
    config.ensure_directories()
    
    app = QApplication(sys.argv)
    app.setApplicationName("QMLauncher")
    app.setOrganizationName("QMProject")
    
    window = MainWindow()
    window.show()
    
    sys.exit(app.exec())

if __name__ == "__main__":
    main()
