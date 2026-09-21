import type { Metadata } from "next";
import { Geist_Mono, Hanken_Grotesk, Outfit, Source_Serif_4 } from "next/font/google";
import "./globals.css";

// Four voices, each with one job: Hanken Grotesk for the interface (warm,
// humanist, never the default), Outfit for the wordmark, Geist Mono for the
// archival stamps (time, domains, counts), Source Serif 4 for the moments
// the app hands you words to read (notes, extracted prose, Jev's answers).
const ui = Hanken_Grotesk({
  subsets: ["latin"],
  variable: "--font-ui",
});

const mark = Outfit({
  subsets: ["latin"],
  variable: "--font-mark",
});

const geist = Geist_Mono({
  subsets: ["latin"],
  variable: "--font-geist",
});

const read = Source_Serif_4({
  subsets: ["latin"],
  variable: "--font-read",
});

export const metadata: Metadata = {
  title: "helmme, personal memory",
  description: "Save anything. Organize nothing. Find everything.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${ui.variable} ${mark.variable} ${geist.variable} ${read.variable}`} suppressHydrationWarning>
      <body className="bg-light font-sans text-deep antialiased">
        <script
          dangerouslySetInnerHTML={{
            // Settled before first paint: saved choice wins, else system.
            __html:
              "(function(){try{var t=localStorage.getItem('helmme-theme');var d=t?t==='dark':window.matchMedia('(prefers-color-scheme: dark)').matches;document.documentElement.classList.toggle('dark',d)}catch(e){}})();",
          }}
        />
        {children}
      </body>
    </html>
  );
}
