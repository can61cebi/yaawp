#pragma once

#include <QObject>

class KStatusNotifierItem;
class QWindow;

// Tray shows a KDE system tray icon and links it to the main window so the
// application keeps running when the window is closed.
class Tray : public QObject
{
    Q_OBJECT

public:
    explicit Tray(QObject *parent = nullptr);

    void setWindow(QWindow *window);

private:
    KStatusNotifierItem *m_sni;
};
