# Achievement Image Generation Design

This file defines the raster assets for the codex achievements on `feat/achievement`.

Current output targets:

- Gallery icons: `frontend/public/game-icons/achievements/*.png`
- Unlock notification illustrations: `frontend/public/game-icons/alerts/*.png`

The repo currently contains placeholder PNGs at every path listed in the asset
matrix. Final generated images should replace those files in place with the
same filenames, so the frontend metadata does not need to change.

The gallery icon assets should be transparent 512x512 PNGs. The notification
illustrations should match the existing alert images: square 1254x1254 scene
illustrations with no embedded text.

## Implementation Hook

Frontend code reads these paths and copy from:

- `frontend/src/features/achievements/lib/achievement-assets.ts`
- `frontend/src/features/achievements/ui/achievement-gallery.tsx`
- `frontend/src/features/reward-alert/ui/reward-alert-center.tsx`

When final art is ready:

1. Replace `frontend/public/game-icons/achievements/achievement-codex-*.png`
   with transparent 512x512 PNGs.
2. Replace `frontend/public/game-icons/alerts/achievement-codex-*-alert.png`
   with 1254x1254 RGB scene PNGs.
3. Keep filenames unchanged.
4. Do not place text inside the images; UI text is rendered from
   `achievement-assets.ts`.

## Shared Style

- Cute semi-3D game illustration.
- Main character is the existing "小石": a small rounded pebble or tilted tablet-stone with a black outline, glossy face panel, simple happy eyes, and tiny arms or feet.
- Strong readable silhouette at mobile card size.
- Rounded chunky forms, warm soft lighting, polished highlights, subtle paper or sticker texture.
- Palette should rotate through existing pebble tones:
  - exploration green
  - inspiration yellow
  - resonance purple
  - engineering blue
  - entertainment orange
- No readable in-image text, no logos, no watermark.
- Use symbolic UI-like objects instead of written labels: shelves, stamps, stars, folders, compasses, maps, collection trays, medals, glowing codex pages.

## Asset Matrix

| Key | Name | Requirement | Reward | Icon File | Notification File |
| --- | --- | ---: | ---: | --- | --- |
| `codex_tier_01` | 石來運轉 | 5 | 500 OP | `achievement-codex-tier-01.png` | `achievement-codex-tier-01-alert.png` |
| `codex_tier_02` | 石在必得 | 10 | 550 OP | `achievement-codex-tier-02.png` | `achievement-codex-tier-02-alert.png` |
| `codex_tier_03` | 一石三鳥 | 15 | 600 OP | `achievement-codex-tier-03.png` | `achievement-codex-tier-03-alert.png` |
| `codex_tier_04` | 與石俱進 | 20 | 650 OP | `achievement-codex-tier-04.png` | `achievement-codex-tier-04-alert.png` |
| `codex_tier_05` | 石半功倍 | 25 | 700 OP | `achievement-codex-tier-05.png` | `achievement-codex-tier-05-alert.png` |
| `codex_tier_06` | 三石而立 | 30 | 750 OP | `achievement-codex-tier-06.png` | `achievement-codex-tier-06-alert.png` |
| `codex_tier_07` | 水滴石穿 | 35 | 800 OP | `achievement-codex-tier-07.png` | `achievement-codex-tier-07-alert.png` |
| `codex_tier_08` | 石破天驚 | 40 | 850 OP | `achievement-codex-tier-08.png` | `achievement-codex-tier-08-alert.png` |
| `codex_tier_09` | 他山之石 | 45 | 900 OP | `achievement-codex-tier-09.png` | `achievement-codex-tier-09-alert.png` |
| `codex_tier_10` | 五石知天命 | 50 | 950 OP | `achievement-codex-tier-10.png` | `achievement-codex-tier-10-alert.png` |
| `codex_complete` | 石全石美 | 52 | 1200 OP | `achievement-codex-complete.png` | `achievement-codex-complete-alert.png` |

## Notification Copy

These are UI strings, not image text.

| Key | Badge | Title | Detail | Reward Line |
| --- | --- | --- | --- | --- |
| `codex_tier_01` | 圖鑑階級 1 | 石來運轉 | 小石把第一批收藏排進展示盤，圖鑑開始轉動了。 | 獲得 500 OP |
| `codex_tier_02` | 圖鑑階級 2 | 石在必得 | 小石確認每個空格都有位置，下一批收藏已經在路上。 | 獲得 550 OP |
| `codex_tier_03` | 圖鑑階級 3 | 一石三鳥 | 小石一次整理三排線索，收藏、回憶和開源力一起入袋。 | 獲得 600 OP |
| `codex_tier_04` | 圖鑑階級 4 | 與石俱進 | 小石替圖鑑裝上更新貼紙，收藏進度又往前了一版。 | 獲得 650 OP |
| `codex_tier_05` | 圖鑑階級 5 | 石半功倍 | 小石把半滿展示櫃擦亮，接下來每一步都更有把握。 | 獲得 700 OP |
| `codex_tier_06` | 圖鑑階級 6 | 三石而立 | 小石立起穩穩的收藏架，三十種小石排成可靠的基座。 | 獲得 750 OP |
| `codex_tier_07` | 圖鑑階級 7 | 水滴石穿 | 小石一格一格補完空位，慢慢收集也能留下清楚痕跡。 | 獲得 800 OP |
| `codex_tier_08` | 圖鑑階級 8 | 石破天驚 | 小石打開高階收藏艙，光從圖鑑縫隙裡冒了出來。 | 獲得 850 OP |
| `codex_tier_09` | 圖鑑階級 9 | 他山之石 | 小石帶回遠方的收藏樣本，圖鑑多了一整段旅程。 | 獲得 900 OP |
| `codex_tier_10` | 圖鑑階級 10 | 五石知天命 | 小石點亮五色收藏環，離完整圖鑑只剩最後一哩。 | 獲得 950 OP |
| `codex_complete` | 完整圖鑑 | 石全石美 | 小石把最後一格放回原位，整本圖鑑終於亮成星圖。 | 獲得 1200 OP |

## Gallery Icon Prompts

Each prompt should generate one transparent 512x512 PNG.

### `achievement-codex-tier-01.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `石來運轉`, representing collecting 5 pebble characters.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: one happy SITCON-style pebble mascot beside a small round display tray holding five tiny colored pebble tokens, with a small compass charm and golden sparkle.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, soft sticker texture.
Composition/framing: centered, full object visible, strong silhouette, generous padding.
Color palette: exploration green with warm gold accents.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-02.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `石在必得`, representing collecting 10 pebble types.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: a confident pebble mascot holding a tiny checklist stamp, in front of a compact wooden collection case with ten gem-like pebble slots.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, soft sticker texture.
Composition/framing: centered badge-like object, case behind mascot, no written labels.
Color palette: inspiration yellow and warm amber.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-03.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `一石三鳥`, representing one collection action unlocking three benefits.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: one pebble mascot launching three glowing star-shaped paper markers from a slingshot-like catalog ribbon, with three small collection tokens orbiting.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, playful motion arcs.
Composition/framing: diagonal action pose, strong silhouette, no written labels.
Color palette: resonance purple with gold sparks.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-04.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `與石俱進`, representing steady codex progress.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: pebble mascot climbing a small stepped progress platform made of codex pages, with an upward arrow charm and updated collection cards.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, polished game icon.
Composition/framing: centered vertical progress shape, clear top-to-bottom movement.
Color palette: engineering blue with teal glow.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-05.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `石半功倍`, representing reaching the halfway collection milestone.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: pebble mascot polishing a half-filled display cabinet; one side has empty silhouettes, the other side has glowing pebble tokens, with a multiplier-like sparkle symbol but no text.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, soft warm lighting.
Composition/framing: symmetrical half-full cabinet, mascot in front.
Color palette: entertainment orange with cream and gold accents.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-06.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `三石而立`, representing 30 collected pebble types and a stable foundation.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: three smiling pebble mascots stacked into a sturdy triangular display stand, holding tiny collection plaques and a stable base.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, soft sticker texture.
Composition/framing: triangular stable silhouette, centered.
Color palette: green, yellow, and blue balanced together.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-07.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `水滴石穿`, representing patient collection progress.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: pebble mascot under a gentle glowing water drop that carves a small star groove into a stone tablet, surrounded by neatly placed collection beads.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy water highlight, soft magical glow.
Composition/framing: calm centered icon with droplet above and tablet below.
Color palette: teal blue, mint green, and silver.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-08.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `石破天驚`, representing a dramatic high-tier codex unlock.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: pebble mascot opening a cracked glowing codex capsule, with light rays, floating shard shapes, and a surprised happy face.
Style/medium: semi-3D kawaii game asset, chunky black outline, energetic glow, polished game icon.
Composition/framing: explosive but controlled central burst, no clutter.
Color palette: deep blue, electric cyan, and gold.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-09.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `他山之石`, representing bringing back collection samples from far away.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: pebble mascot with a small telescope and folded map, carrying a satchel of distant mountain-shaped pebble samples.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, travel souvenir feeling.
Composition/framing: mascot angled slightly, telescope and map readable as objects.
Color palette: purple, moss green, and warm parchment.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-tier-10.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `五石知天命`, representing 50 collected pebble types and five color families.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: central pebble mascot holding a glowing compass-codex medallion, surrounded by five colored pebble family tokens forming a ring.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, ceremonial milestone icon.
Composition/framing: circular ring composition, clear central medallion.
Color palette: five pebble colors: green, yellow, purple, blue, orange, with gold.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

### `achievement-codex-complete.png`

Use case: stylized-concept
Asset type: mobile game achievement gallery icon, transparent PNG
Primary request: Create a cute collectible achievement icon for `石全石美`, representing complete collection of all pebble types.
Scene/backdrop: perfectly flat solid #00ff00 chroma-key background for background removal; no shadows, gradients, texture, floor plane, or lighting variation.
Subject: triumphant pebble mascot in front of a fully lit codex display cabinet with five-color stones, a small crown-like star ornament, and radiant completion glow.
Style/medium: semi-3D kawaii game asset, chunky black outline, glossy highlights, premium completion badge.
Composition/framing: centered celebratory badge, rich but readable silhouette.
Color palette: gold, white, five pebble colors, soft rainbow highlight.
Constraints: no text, no logo, no watermark, do not use #00ff00 anywhere in the subject, 512x512, readable at 96px.

## Notification Illustration Prompts

Each prompt should generate one square 1254x1254 RGB PNG. Do not embed text in
the image. The UI will render badge/title/detail/reward separately.

### `achievement-codex-tier-01-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A happy pebble mascot starts a small codex display tray after collecting the first five pebble types.
Scene/backdrop: cozy camp archive corner at night, warm lanterns, small shelves, a five-slot display tray glowing softly.
Subject: central pebble mascot placing the fifth colored pebble token into the tray, smiling with excitement.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: square scene, mascot large in foreground, display tray clearly visible, no text.
Lighting/mood: warm celebratory glow, gentle sparkles.
Color palette: green and gold.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-02-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot confidently organizes ten collected pebble tokens into a portable collection case.
Scene/backdrop: neat workbench with drawers, soft desk lamp, small catalog tools.
Subject: central pebble mascot stamping a collection case with ten glowing slots, all objects symbolic only.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: square scene, case open toward camera, no text.
Lighting/mood: warm focused achievement mood.
Color palette: yellow, amber, cream.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-03-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot celebrates a triple collection milestone, with three glowing collection markers flying into place.
Scene/backdrop: playful camp gallery with floating shelves and tiny star lights.
Subject: mascot pulls a ribbon attached to a codex board, sending three star-shaped markers into three empty display slots.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: dynamic diagonal motion, mascot foreground, three markers clearly visible, no text.
Lighting/mood: playful motion, purple celebration glow.
Color palette: purple, gold, soft blue.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-04-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot upgrades a codex progress machine after reaching twenty collected pebble types.
Scene/backdrop: compact tech archive room with cables, blue screens, and display shelves.
Subject: mascot climbing a little step ladder to plug a glowing progress charm into a codex console.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: square scene, vertical progress feeling, no text.
Lighting/mood: cool tech glow with warm fill light.
Color palette: engineering blue, teal, soft yellow.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-05-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot polishes a half-filled codex display cabinet after reaching the halfway milestone.
Scene/backdrop: cozy museum-like camp room with a split display cabinet, one side lit, one side waiting.
Subject: mascot holding a cloth and tiny brush, proudly looking at the glowing half of the collection.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: display cabinet behind mascot, no text.
Lighting/mood: warm orange light, clean satisfying milestone.
Color palette: orange, cream, gold.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-06-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: Three pebble mascots build a stable triangular collection stand after reaching thirty collected pebble types.
Scene/backdrop: camp workshop floor with soft rugs, shelves of pebble tokens, construction tools.
Subject: three mascots cooperating to hold up a sturdy triangular display shelf filled with tokens.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: triangular shelf in center, teamwork scene, no text.
Lighting/mood: cheerful teamwork, warm and balanced.
Color palette: green, yellow, blue, wood tones.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-07-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot patiently completes another codex row while a magical water drop carves a tiny star groove.
Scene/backdrop: quiet night archive with a small fountain-like collection table, soft plants and lights.
Subject: mascot watches a glowing droplet land on a stone tablet beside neatly ordered pebble tokens.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: calm centered scene, droplet and tablet clearly visible, no text.
Lighting/mood: gentle perseverance, teal glow.
Color palette: teal, mint, silver, warm lantern highlights.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-08-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot opens a high-tier codex capsule and bright light bursts out after reaching forty collected pebble types.
Scene/backdrop: dramatic camp archive vault with rounded machinery and floating shards.
Subject: central mascot surprised and happy, holding open a glowing capsule filled with collection tokens.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: energetic burst in center, readable mascot expression, no text.
Lighting/mood: dramatic but friendly, cyan and gold light rays.
Color palette: deep blue, cyan, gold.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-09-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot returns from a collection journey with distant stone samples after reaching forty-five collected pebble types.
Scene/backdrop: camp map room with window view of distant stylized mountains, travel souvenirs, and soft lanterns.
Subject: mascot wearing a small scarf, holding telescope and map, unloading rare pebble samples into a collection tray.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: mascot foreground, map room and mountain window behind, no text.
Lighting/mood: adventurous, warm evening light.
Color palette: purple, moss green, parchment, gold.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-tier-10-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot lights a five-color codex ring after reaching fifty collected pebble types.
Scene/backdrop: ceremonial archive chamber with circular display table and five color families of pebble tokens.
Subject: central mascot holding a glowing compass-codex medallion while five colored token groups form a ring around it.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images.
Composition/framing: circular composition, milestone feeling, no text.
Lighting/mood: bright ceremonial glow, anticipation before completion.
Color palette: five pebble tones with gold and white highlights.
Constraints: no readable text, no logo, no watermark, no UI overlays.

### `achievement-codex-complete-alert.png`

Use case: illustration-story
Asset type: achievement unlock notification illustration
Primary request: A pebble mascot completes the entire codex collection; every slot in the display cabinet lights up like a star map.
Scene/backdrop: grand cozy camp archive, fully lit collection wall, lanterns, star-shaped lights, soft plants and polished floor.
Subject: triumphant central pebble mascot placing the final token into a complete five-color collection cabinet, with joyful sparkles.
Style/medium: cute semi-3D game illustration matching existing SITCON Camp alert images, premium final achievement moment.
Composition/framing: mascot large in foreground, complete cabinet visible behind, no text.
Lighting/mood: warm finale celebration, magical gold and rainbow glow.
Color palette: gold, white, green, yellow, purple, blue, orange.
Constraints: no readable text, no logo, no watermark, no UI overlays.
