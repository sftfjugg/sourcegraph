import { createContext, useContext } from 'react'

import { TabsSettings } from '.'

export const TabsSettingsContext = createContext<TabsSettings | null>(null)
TabsSettingsContext.displayName = 'TabsSettingsContext'

export const useTabsSettings = (): TabsSettings => {
    const context = useContext(TabsSettingsContext)
    if (!context) {
        throw new Error('useTabsSettingsContext or Tabs inner components cannot be used outside <Tabs> sub-tree')
    }

    return context
}

export const TabPanelIndexContext = createContext<number>(0)
TabPanelIndexContext.displayName = 'TabPanelIndexContext'

export const useTablePanelIndex = (): number => {
    const context = useContext(TabPanelIndexContext)
    if (context === undefined) {
        throw new Error('TabPanelIndexContext cannot be used outside <Tabs> sub-tree')
    }
    return context
}
