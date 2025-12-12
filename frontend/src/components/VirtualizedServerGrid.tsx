/**
 * Виртуализированная сетка серверов для оптимизации рендеринга больших списков
 * Использует react-window для виртуализации
 */

import { FixedSizeGrid as Grid } from 'react-window'
import { Server as ServerType } from '../shared/types'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Play, Download, Users, Trash2, Info, Settings } from 'lucide-react'
import { useI18n } from '../contexts/I18nContext'

interface VirtualizedServerGridProps {
  servers: (ServerType & { 
    embedded?: any
    players_online?: number
    players_max?: number
    clientInstalled?: boolean
  })[]
  columns?: number
  itemWidth?: number
  itemHeight?: number
  onServerClick?: (server: ServerType) => void
  onInstall?: (server: ServerType) => void
  onLaunch?: (server: ServerType) => void
  onUninstall?: (server: ServerType) => void
  onSettings?: (server: ServerType) => void
  onInfo?: (server: ServerType) => void
  isInstalling?: (serverId: number) => boolean
  isLaunching?: boolean
}

export function VirtualizedServerGrid({
  servers,
  columns = 3,
  itemWidth = 350,
  itemHeight = 400,
  onServerClick,
  onInstall,
  onLaunch,
  onUninstall,
  onSettings,
  onInfo,
  isInstalling,
  isLaunching,
}: VirtualizedServerGridProps) {
  const { t } = useI18n()
  const rowCount = Math.ceil(servers.length / columns)

  const Cell = ({ columnIndex, rowIndex, style }: any) => {
    const index = rowIndex * columns + columnIndex
    if (index >= servers.length) return null

    const server = servers[index]

    return (
      <div style={style} className="p-3">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: index * 0.05 }}
          whileHover={{ y: -4 }}
        >
          <Card className="group relative hover:shadow-lg transition-all cursor-pointer h-full">
            {/* Server Image */}
            <div className="relative w-full h-48 overflow-hidden bg-muted">
              <img
                src={server.preview_image_url || '/minecraft-server-preview.svg'}
                alt={server.name}
                className="w-full h-full object-cover group-hover:scale-110 transition-transform duration-500"
                onError={(e) => {
                  e.currentTarget.style.display = 'none'
                  const parent = e.currentTarget.parentElement!
                  if (!parent.querySelector('.fallback')) {
                    parent.innerHTML = '<div class="fallback w-full h-full flex items-center justify-center text-6xl">🎮</div>'
                  }
                }}
              />
            </div>

            <CardHeader>
              <CardTitle className="line-clamp-1">
                {server.server_name || server.name}
              </CardTitle>
              {server.server_address && (
                <CardDescription className="font-mono text-xs">
                  {server.server_address}:{server.server_port}
                </CardDescription>
              )}
            </CardHeader>

            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <Badge variant="secondary">
                  MC {server.minecraft_version || 'N/A'}
                </Badge>
                {server.players_online !== undefined && (
                  <div className="flex items-center gap-1.5">
                    <Users className="w-4 h-4 text-muted-foreground" />
                    <span className="text-xs font-medium text-muted-foreground">
                      {server.players_online || 0}/{server.players_max || 0}
                    </span>
                  </div>
                )}
              </div>

              {server.description && (
                <CardDescription className="line-clamp-2">
                  {server.description}
                </CardDescription>
              )}

              <div className="flex gap-2 pt-4">
                {server.clientInstalled ? (
                  <Button
                    className="flex-1"
                    onClick={(e) => {
                      e.stopPropagation()
                      onLaunch?.(server)
                    }}
                    disabled={isLaunching}
                  >
                    <Play className="w-4 h-4 mr-2" />
                    {t('servers.play') || 'Играть'}
                  </Button>
                ) : (
                  <Button
                    className="flex-1"
                    onClick={(e) => {
                      e.stopPropagation()
                      onInstall?.(server)
                    }}
                    disabled={isInstalling?.(server.id)}
                  >
                    <Download className="w-4 h-4 mr-2" />
                    {t('servers.install') || 'Установить'}
                  </Button>
                )}
                <Button
                  variant="outline"
                  size="icon"
                  onClick={(e) => {
                    e.stopPropagation()
                    onSettings?.(server)
                  }}
                >
                  <Settings className="w-4 h-4" />
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={(e) => {
                    e.stopPropagation()
                    onInfo?.(server)
                  }}
                >
                  <Info className="w-4 h-4" />
                </Button>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    )
  }

  if (servers.length === 0) {
    return (
      <div className="text-center py-20">
        <p className="text-xl text-foreground mb-2">Нет доступных серверов</p>
        <p className="text-muted-foreground">Создайте сервер в QMAdmin для отображения здесь</p>
      </div>
    )
  }

  return (
    <Grid
      columnCount={columns}
      columnWidth={itemWidth}
      height={600} // Высота контейнера
      rowCount={rowCount}
      rowHeight={itemHeight}
      width={columns * itemWidth}
      className="no-drag"
    >
      {Cell}
    </Grid>
  )
}

