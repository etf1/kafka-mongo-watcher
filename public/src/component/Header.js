import { html } from 'https://unpkg.com/htm@3.1.0/preact/standalone.module.js'

export const Header = () => {
    return html`
        <div class="bg-gray-800 text-white font-sans w-full m-0">
            <div class="shadow-xl">
                <div class="container mx-auto px-4">
                    <div class="flex items-center justify-between py-2">
                        <div class="flex items-center">
                            <a href="/" class="text-sm font-semibold hover:text-gray-100 mr-4 uppercase">Debug UI</a>
                        </div>

                        <div class="flex items-center">
                            <a href="https://github.com/etf1/kafka-mongo-watcher" target="_blank" class="text-sm font-semibold border px-4 py-2 rounded-lg hover:text-gray-800 hover:border-gray-800 hover:bg-white">GitHub</a>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="w-full my-2">
            <div class="p-2 mx-6 text-white text-xs">
                <div class="flex flex-row place-content-center gap-4">
                    <div class="bg-gray-800 px-3 py-2 rounded-full">
                        <span>Database:</span> <strong>${window.__context.database}</strong>
                    </div>
                    <div class="bg-gray-800 px-3 py-2 rounded-full">
                        <span>Collection:</span> <strong>${window.__context.collection}</strong>
                    </div>
                </div>
            </div>
        </div>
    `
}