import { LoginPanel, StoneGlyph, stoneTypes } from "@/features/login"

export function LoginPage() {
  return (
    <main className="login-mobile-stage" aria-label="Camp 2026 Game 登入頁">
      <div className="login-map-lines" aria-hidden="true" />
      {stoneTypes.map((type) => (
        <span
          className="floating-stone"
          key={type.name}
          style={{
            left: type.x,
            top: type.y,
            transform: `rotate(${type.rotate})`,
          }}
          aria-hidden="true"
        >
          <StoneGlyph type={type} tiny />
        </span>
      ))}

      <div className="login-screen-topbar" aria-hidden="true">
        <span>20:26</span>
        <span className="signal-dots">
          <i />
          <i />
          <i />
        </span>
      </div>

      <LoginPanel />

      <footer className="login-footer">
        <span>Field Kit Collectible UI</span>
        <span>無漸層・扁平實色</span>
      </footer>
    </main>
  )
}
