import { useEffect, useRef } from "react";
import * as skinview3d from "skinview3d";

interface SkinPreview3dProps {
  skinUrl: string;
  skinModel?: "steve" | "alex";
  width?: number;
  height?: number;
  className?: string;
}

export function SkinPreview3d({
  skinUrl,
  skinModel = "steve",
  width = 200,
  height = 280,
  className = "",
}: SkinPreview3dProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const viewerRef = useRef<skinview3d.SkinViewer | null>(null);

  useEffect(() => {
    if (!canvasRef.current) return;

    const viewer = new skinview3d.SkinViewer({
      canvas: canvasRef.current,
      width,
      height,
    });
    viewerRef.current = viewer;

    return () => {
      viewer.dispose();
      viewerRef.current = null;
    };
  }, []);

  useEffect(() => {
    const viewer = viewerRef.current;
    if (!viewer) return;

    if (skinUrl) {
      viewer.loadSkin(skinUrl, { model: skinModel === "alex" ? "slim" : "default" });
    } else {
      viewer.loadSkin(null);
    }
  }, [skinUrl, skinModel]);

  useEffect(() => {
    const viewer = viewerRef.current;
    if (!viewer) return;
    viewer.width = width;
    viewer.height = height;
  }, [width, height]);

  return (
    <canvas
      ref={canvasRef}
      className={`rounded-lg border border-border bg-muted/30 ${className}`}
      style={{ width, height }}
      title="Перетащите для поворота"
    />
  );
}
