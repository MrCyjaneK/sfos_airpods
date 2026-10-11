#ifdef QT_QML_DEBUG
#include <QtQuick>
#endif

#include <QGuiApplication>
#include <QQuickView>
#include <sailfishapp.h>

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> app(SailfishApp::application(argc, argv));
    app->setOrganizationName(QStringLiteral("net.mrcyjanek"));
    app->setApplicationName(QStringLiteral("sfos_airpods"));
    QScopedPointer<QQuickView> view(SailfishApp::createView());
    view->setSource(SailfishApp::pathToMainQml());
    view->show();
    return app->exec();
}
