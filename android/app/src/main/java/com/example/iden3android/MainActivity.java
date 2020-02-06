package com.example.iden3android;

import android.os.Bundle;

import com.google.android.material.floatingactionbutton.FloatingActionButton;
import com.google.android.material.snackbar.Snackbar;

import androidx.appcompat.app.AppCompatActivity;
import androidx.appcompat.widget.Toolbar;

import android.util.Log;
import android.view.View;
import android.view.Menu;
import android.view.MenuItem;
import android.widget.TextView;

import iden3mobile.Identity;

public class MainActivity extends AppCompatActivity {
    Identity iden3;
    CallbackHandler callback;
    String serverIP = "192.168.68.120";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        // Init view
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);

        // Main text instance
        final TextView mainText = findViewById(R.id.mainText);
        callback = new CallbackHandler(mainText);

        // Init iden3
        iden3 = new Identity();
        try {
            iden3.createIdentity();
        } catch (Exception e) {
            e.printStackTrace();
        }
        // String storagePath = getFilesDir().getAbsolutePath();
        // iden3.setPath(storagePath);
        iden3.setCallbackHandler(callback);

        // Action button (call a go function)
        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                String ticket = iden3.requestClaim("http://"+serverIP+":1234/issueClaim");
                Log.println(Log.ERROR,"MAIN: requestClaim", ticket);
                mainText.setText("\nWaiting issuer to buidl claim:" + ticket);
            }
        });

        // Still alive button (just to check that the app is not blocked, for async testing)
        FloatingActionButton live = findViewById(R.id.live);
        live.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                Snackbar.make(view, "still alive", Snackbar.LENGTH_LONG)
                        .setAction("Action", null).show();
            }
        });
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        // Inflate the menu; this adds items to the action bar if it is present.
        getMenuInflater().inflate(R.menu.menu_main, menu);
        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        // Handle action bar item clicks here. The action bar will
        // automatically handle clicks on the Home/Up button, so long
        // as you specify a parent activity in AndroidManifest.xml.
        int id = item.getItemId();

        //noinspection SimplifiableIfStatement
        if (id == R.id.action_settings) {
            return true;
        }

        return super.onOptionsItemSelected(item);
    }
}
