import React, { createContext, useContext, useState, ReactNode } from 'react';

interface AppStateContextType {
  showDrawerToggle: boolean;
  setShowDrawerToggle: (show: boolean) => void;
}

const AppStateContext = createContext<AppStateContextType | undefined>(undefined);

export function AppStateProvider({ children }: { children: ReactNode }) {
  const [showDrawerToggle, setShowDrawerToggle] = useState(false);

  return (
    <AppStateContext.Provider value={{ showDrawerToggle, setShowDrawerToggle }}>
      {children}
    </AppStateContext.Provider>
  );
}

export function useAppState() {
  const context = useContext(AppStateContext);
  if (context === undefined) {
    throw new Error('useAppState must be used within an AppStateProvider');
  }
  return context;
}
