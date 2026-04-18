import React, { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "@/components/ui/button";

type Props = { children: ReactNode };

type State = { error: Error | null };

/** Catches render errors so the WebView is not a blank white screen with no hint. */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[QMLauncher UI]", error, info.componentStack);
  }

  handleRetry = () => {
    this.setState({ error: null });
  };

  render() {
    if (this.state.error) {
      return (
        <div className="min-h-svh flex flex-col items-center justify-center bg-background text-foreground p-6">
          <div className="w-full max-w-lg space-y-4 rounded-xl border border-border bg-card p-6 shadow-sm">
            <h1 className="text-lg font-semibold">Ошибка интерфейса</h1>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Сообщение ниже можно отправить разработчику. Кнопка «Сбросить» пробует снова отрисовать
              окно без перезапуска приложения.
            </p>
            <pre className="max-h-48 overflow-auto rounded-md border border-border bg-muted/50 p-3 text-xs whitespace-pre-wrap break-words">
              {this.state.error.message}
              {"\n\n"}
              {this.state.error.stack}
            </pre>
            <div className="flex flex-wrap gap-2">
              <Button type="button" variant="default" onClick={this.handleRetry}>
                Сбросить экран
              </Button>
              <Button type="button" variant="outline" onClick={() => window.location.reload()}>
                Полная перезагрузка
              </Button>
            </div>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
