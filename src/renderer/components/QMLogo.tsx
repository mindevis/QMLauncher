import qmLogoUrl from '../assets/qm-logo.png'

export function QMLogo({ className }: { className?: string }) {
  return (
    <img 
      src={qmLogoUrl} 
      alt="QM Logo" 
      className={className}
    />
  )
}

