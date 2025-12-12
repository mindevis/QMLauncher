import { Newspaper } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { useI18n } from '../contexts/I18nContext'

export function NewsTab() {
  const { t } = useI18n()
  
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto no-drag">
        <h2 className="text-3xl font-bold text-foreground mb-6 flex items-center gap-3">
          <Newspaper className="w-8 h-8" />
          {t('news.title')}
        </h2>
        
        <div className="space-y-4">
          <Card className="hover:shadow-lg transition-all">
            <CardHeader>
              <CardTitle>{t('news.welcomeTitle')}</CardTitle>
              <CardDescription>25.11.2024</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">
                {t('news.welcomeDescription')}
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
