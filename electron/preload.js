// This file will be generated from preload.ts during build
// For development, we'll use a simple bridge
if (typeof window !== 'undefined') {
  window.electronAPI = {
    getAppVersion: () => Promise.resolve('1.0.0'),
    getPlatform: () => Promise.resolve(process.platform),
    getLauncherConfig: () => Promise.resolve({
      javaPath: 'java',
      minMemory: 1024,
      maxMemory: 4096,
      windowWidth: 1200,
      windowHeight: 800,
      selectedProfile: null,
      profiles: {}
    }),
    saveLauncherConfig: () => Promise.resolve({ success: true }),
    launchMinecraft: () => Promise.resolve({ success: false, error: 'Not implemented' }),
    stopMinecraft: () => Promise.resolve({ success: false, error: 'Not implemented' }),
    onMinecraftExited: () => {},
    onMinecraftError: () => {}
  }
}

